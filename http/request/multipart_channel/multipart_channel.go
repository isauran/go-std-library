package main

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
)

// signalReader wraps io.Reader and signals on first read
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

type Multipart struct {
	client    *http.Client
	req       *http.Request
	wg        sync.WaitGroup
	mw        *multipart.Writer
	pw        *io.PipeWriter
	bodyCh    chan request
	startedCh chan struct{}
}

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

func (m *Multipart) Param(key, value string) *Multipart {
	m.bodyCh <- request{reqType: stringType, key: key, value: value}
	return m
}

func (m *Multipart) Bool(key string, value bool) *Multipart {
	return m.Param(key, strconv.FormatBool(value))
}

func (m *Multipart) Float(key string, value float64) *Multipart {
	return m.Param(key, strconv.FormatFloat(value, 'f', -1, 64))
}

func (m *Multipart) Int(key string, value int) *Multipart {
	return m.Param(key, strconv.Itoa(value))
}

func (m *Multipart) File(key, filename string, content io.Reader) *Multipart {
	m.bodyCh <- request{reqType: fileType, key: key, value: filename, content: content}
	return m
}

func (m *Multipart) Header(key, value string) *Multipart {
	m.req.Header.Set(key, value)
	return m
}

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
