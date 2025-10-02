package main

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
)

type Reader struct {
	reader       io.Reader
	once         sync.Once
	startRequest chan struct{}
}

func (s *Reader) Read(p []byte) (n int, err error) {
	s.once.Do(func() {
		close(s.startRequest)
	})
	return s.reader.Read(p)
}

type RequestType int

const (
	NoneType RequestType = iota
	StringType
	FileType
	JSONType
)

type TRequest struct {
	Type    RequestType
	Key     string
	Value   string
	Content io.Reader
}

type Multipart struct {
	client       *http.Client
	request      *http.Request
	wg           sync.WaitGroup
	mw           *multipart.Writer
	pr           *io.PipeReader
	pw           *io.PipeWriter
	body         chan TRequest
	resp         chan *http.Response
	err          chan error
	startRequest chan struct{}
}

func NewMultipart(ctx context.Context, client *http.Client, method, url string) *Multipart {
	pipeReader, pipeWriter := io.Pipe()
	ch := make(chan TRequest, 100)
	r := &Multipart{
		client:       client,
		body:         ch,
		pr:           pipeReader,
		pw:           pipeWriter,
		mw:           multipart.NewWriter(pipeWriter),
		resp:         make(chan *http.Response, 1),
		err:          make(chan error, 1),
		startRequest: make(chan struct{}),
	}

	r.request, _ = http.NewRequestWithContext(ctx, method, url, &Reader{reader: pipeReader, startRequest: r.startRequest})
	r.request.Header.Set("Content-Type", r.mw.FormDataContentType())

	// Start worker that will write to pipe
	r.wg.Add(1)
	go r.worker()

	return r
}

func (r *Multipart) worker() {
	defer r.wg.Done()
	for b := range r.body {
		switch b.Type {
		case StringType:
			{
				err := r.mw.WriteField(b.Key, b.Value)
				if err != nil {
					r.pw.CloseWithError(fmt.Errorf("failed to write form field [%q] value %s: %w", b.Key, b.Value, err))
					return
				}
			}
		case FileType:
			{
				part, err := r.mw.CreateFormFile(b.Key, b.Value)
				if err != nil {
					r.pw.CloseWithError(fmt.Errorf("failed to create form file: %w", err))
					return
				}
				if _, err := io.Copy(part, b.Content); err != nil {
					r.pw.CloseWithError(fmt.Errorf("failed to copy file content: %w", err))
					return
				}
			}
		}
	}
}

func (r *Multipart) Param(key, value string) *Multipart {
	r.body <- TRequest{Type: StringType, Key: key, Value: value}
	return r
}

func (r *Multipart) Bool(fieldName string, value bool) *Multipart {
	return r.Param(fieldName, strconv.FormatBool(value))
}

func (r *Multipart) Float(fieldName string, value float64) *Multipart {
	return r.Param(fieldName, strconv.FormatFloat(value, 'f', -1, 64))
}

func (r *Multipart) File(key, filename string, content io.Reader) *Multipart {
	r.body <- TRequest{Type: FileType, Key: key, Value: filename, Content: content}
	return r
}

func (r *Multipart) Header(key, value string) *Multipart {
	r.request.Header.Set(key, value)
	return r
}

func (r *Multipart) close() {
	close(r.body)
	r.wg.Wait()
	r.mw.Close()
	r.pw.Close()
}

func (r *Multipart) Send() (*http.Response, error) {
	// Start HTTP request with all headers set
	go func() {
		resp, err := r.client.Do(r.request)
		if err != nil {
			r.err <- err
			return
		}
		r.resp <- resp
	}()

	// Wait for HTTP client to start reading from pipe
	<-r.startRequest

	// Close to signal worker to finish and wait
	r.close()

	// Wait for HTTP response
	select {
	case resp := <-r.resp:
		return resp, nil
	case err := <-r.err:
		return nil, err
	}
}
