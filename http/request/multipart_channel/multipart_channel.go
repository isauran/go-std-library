package main

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
)

var _ io.Reader = (*signalReader)(nil)

type signalReader struct {
	reader    io.Reader
	once      sync.Once
	startedCh chan struct{}
}

func (r *signalReader) Read(p []byte) (n int, err error) {
	r.once.Do(func() {
		close(r.startedCh)
	})
	return r.reader.Read(p)
}

type requestType int

const (
	stringType requestType = iota
	fileType
)

type request struct {
	reqType    requestType
	key, value string
	content    io.Reader
}

// Multipart provides a streaming multipart/form-data builder for HTTP requests.
// It uses io.Pipe to stream data without buffering the entire payload in memory.
// All methods (except Send) are chainable for fluent API usage.
type Multipart struct {
	client    *http.Client
	req       *http.Request
	wg        sync.WaitGroup
	mw        *multipart.Writer
	pw        *io.PipeWriter
	bodyCh    chan request
	startedCh chan struct{}
}

// NewMultipart creates a new Multipart builder for streaming multipart/form-data requests.
// The worker goroutine starts immediately and waits for data to be added via Param/File methods.
// Call Send to execute the HTTP request with all accumulated headers and form data.
func NewMultipart(ctx context.Context, client *http.Client, method, url string) *Multipart {
	pr, pw := io.Pipe()
	started := make(chan struct{})

	m := &Multipart{
		client:    client,
		bodyCh:    make(chan request, 100),
		pw:        pw,
		mw:        multipart.NewWriter(pw),
		startedCh: started,
	}

	reader := &signalReader{reader: pr, startedCh: started}
	// Error from NewRequestWithContext is ignored as it only fails for invalid method/URL
	m.req, _ = http.NewRequestWithContext(ctx, method, url, reader)
	m.req.Header.Set("Content-Type", m.mw.FormDataContentType())

	m.wg.Add(1)
	go m.worker()

	return m
}

func (m *Multipart) worker() {
	defer m.wg.Done()
	for req := range m.bodyCh {
		var err error
		switch req.reqType {
		case stringType:
			err = m.mw.WriteField(req.key, req.value)
		case fileType:
			var part io.Writer
			part, err = m.mw.CreateFormFile(req.key, req.value)
			if err == nil {
				_, err = io.Copy(part, req.content)
			}
		}
		if err != nil {
			m.pw.CloseWithError(err)
			return
		}
	}
}

// Param adds a string field to the multipart form.
// Returns the Multipart instance for method chaining.
func (m *Multipart) Param(key, value string) *Multipart {
	m.bodyCh <- request{reqType: stringType, key: key, value: value}
	return m
}

// Bool adds a boolean field to the multipart form.
// The value is converted to "true" or "false" string.
func (m *Multipart) Bool(key string, value bool) *Multipart {
	return m.Param(key, strconv.FormatBool(value))
}

// Float adds a float64 field to the multipart form.
// The value is formatted using strconv.FormatFloat with 'f' format.
func (m *Multipart) Float(key string, value float64) *Multipart {
	return m.Param(key, strconv.FormatFloat(value, 'f', -1, 64))
}

// Int adds an integer field to the multipart form.
func (m *Multipart) Int(key string, value int) *Multipart {
	return m.Param(key, strconv.Itoa(value))
}

// File adds a file field to the multipart form.
// The content is read from the provided io.Reader and streamed to the request.
func (m *Multipart) File(key, filename string, content io.Reader) *Multipart {
	m.bodyCh <- request{reqType: fileType, key: key, value: filename, content: content}
	return m
}

// Header sets an HTTP header on the request.
// This should be called before Send to ensure headers are included in the request.
func (m *Multipart) Header(key, value string) *Multipart {
	m.req.Header.Set(key, value)
	return m
}

// Send executes the HTTP request with all accumulated form data and headers.
// It waits for the HTTP client to start reading, then closes the form writer
// and waits for the response. Returns the HTTP response or an error.
func (m *Multipart) Send() (*http.Response, error) {
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)

	go func() {
		resp, err := m.client.Do(m.req)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	<-m.startedCh

	close(m.bodyCh)
	m.wg.Wait()
	m.mw.Close()
	m.pw.Close()

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}
