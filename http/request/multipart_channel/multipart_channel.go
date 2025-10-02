package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

type Data struct {
	Type  string
	Value any
}

type Request struct {
	client  *http.Client
	request *http.Request
	ch      chan Data
	wg      sync.WaitGroup
	mw      *multipart.Writer
	pr      *io.PipeReader
	pw      *io.PipeWriter
	respCh  chan *http.Response
	errCh   chan error
}

func NewRequest(client *http.Client, url string) *Request {
	pipeReader, pipeWriter := io.Pipe()
	ch := make(chan Data) // Unbuffered channel to preserve the order of operations.
	r := &Request{
		client: client,
		ch:     ch,
		pr:     pipeReader,
		pw:     pipeWriter,
		mw:     multipart.NewWriter(pipeWriter),
		respCh: make(chan *http.Response, 1),
		errCh:  make(chan error, 1),
	}

	// Create HTTP request with pipe reader
	r.request, _ = http.NewRequest("POST", url, pipeReader)
	r.request.Header.Set("Content-Type", r.mw.FormDataContentType())

	// Start worker that will write to pipe
	r.wg.Add(1)
	go r.worker()

	// Start HTTP request in background immediately
	go func() {
		resp, err := r.client.Do(r.request)
		if err != nil {
			r.errCh <- err
			return
		}
		r.respCh <- resp
	}()

	// Give HTTP client time to start
	time.Sleep(50 * time.Millisecond)

	return r
}

func (r *Request) worker() {
	defer r.wg.Done()
	for data := range r.ch {
		if data.Type == "string" {
			if str, ok := data.Value.(string); ok {
				err := r.mw.WriteField("string", str)
				if err != nil {
					fmt.Println("Error writing field:", err)
					continue
				}
			}
		} else if data.Type == "json" {
			part, err := r.mw.CreateFormFile("json", "data.json")
			if err != nil {
				fmt.Println("Error creating form file:", err)
				continue
			}
			jsonData, err := json.Marshal(data.Value)
			if err != nil {
				fmt.Println("Error marshaling JSON:", err)
				continue
			}
			_, err = part.Write(jsonData)
			if err != nil {
				fmt.Println("Error writing to part:", err)
				continue
			}
		}
	}
}

func (r *Request) String(line string) *Request {
	r.ch <- Data{Type: "string", Value: line}
	return r
}

func (r *Request) JSON(j any) *Request {
	r.ch <- Data{Type: "json", Value: j}
	return r
}

func (r *Request) Header(key, value string) *Request {
	r.request.Header.Set(key, value)
	return r
}

func (r *Request) Close() {
	close(r.ch)
	r.wg.Wait()
	r.mw.Close()
	r.pw.Close()
}

func (r *Request) Send() (*http.Response, error) {
	// Close to signal worker to finish and wait
	r.Close()

	// Wait for HTTP response
	select {
	case resp := <-r.respCh:
		return resp, nil
	case err := <-r.errCh:
		return nil, err
	}
}

func main() {
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/upload", uploadHandler)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	client := http.DefaultClient

	resp, err := NewRequest(client, "http://localhost:8080/upload").
		Header("X-Custom-Header", "custom-value").
		Header("Authorization", "Bearer token123").
		String("1").
		String("2").
		String("3").
		JSON(map[string]string{"key": "value"}).
		Send()

	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}
	fmt.Printf("Response: %s\n", body)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown error: %v\n", err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log received headers
	fmt.Println("=== Received Headers ===")
	for key, values := range r.Header {
		for _, value := range values {
			fmt.Printf("Header: %s = %s\n", key, value)
		}
	}
	fmt.Println("========================")

	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "Received multipart form:\n")
	fmt.Fprintf(w, "\nHeaders:\n")
	fmt.Fprintf(w, "  X-Custom-Header: %s\n", r.Header.Get("X-Custom-Header"))
	fmt.Fprintf(w, "  Authorization: %s\n", r.Header.Get("Authorization"))
	fmt.Fprintf(w, "\n")

	// Handle form fields
	for key, values := range r.MultipartForm.Value {
		for _, value := range values {
			fmt.Fprintf(w, "Field %s: %s\n", key, value)
		}
	}

	// Handle files
	for key, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "File %s (%s): %s\n", key, fileHeader.Filename, string(content))
		}
	}
}
