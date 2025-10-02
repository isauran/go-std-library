package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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

	html := strings.NewReader("<html><body><h1>Hello World!</h1></body></html>")

	resp, err := NewMultipart(context.Background(), client, http.MethodPost, "http://localhost:8080/upload").
		Header("X-Custom-Header", "custom-value").
		Header("Authorization", "Bearer token123").
		Param("key1", "1").
		Param("key2", "2").
		Param("key3", "3").
		File("file", "hello.html", html).
		Param("key4", "4").
		Header("X-Custom-Header2", "123").
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
