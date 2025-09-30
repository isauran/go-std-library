package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func main() {
	fmt.Println("=== Streaming Multipart HTTP Request Demo ===")
	fmt.Println()

	// Example of streaming multipart for large files
	streamingMultipartExample()
}

// streamingMultipartExample demonstrates using io.Pipe for streaming multipart
func streamingMultipartExample() {
	// Create pipe for streaming
	pr, pw := io.Pipe()

	// Create HTTP request with reader part of pipe
	req, err := http.NewRequest("POST", "https://httpbin.org/post", pr)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Create multipart writer for writing to pipe
	mw := multipart.NewWriter(pw)

	// Set proper Content-Type
	req.Header.Set("Content-Type", mw.FormDataContentType())

	fmt.Printf("Content-Type: %s\n", req.Header.Get("Content-Type"))
	fmt.Println("Starting streaming request...")

	// Launch goroutine for writing multipart data
	go func() {
		defer pw.Close()
		defer mw.Close()

		// Add text fields
		if err := mw.WriteField("title", "Large Report"); err != nil {
			pw.CloseWithError(fmt.Errorf("error writing title field: %w", err))
			return
		}

		if err := mw.WriteField("type", "streaming"); err != nil {
			pw.CloseWithError(fmt.Errorf("error writing type field: %w", err))
			return
		}

		// Create file field for "large" file
		fileWriter, err := mw.CreateFormFile("large_file", "big_report.txt")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("error creating file field: %w", err))
			return
		}

		// Simulate large file by writing data in blocks
		baseContent := "Data line in large file. "
		for i := 0; i < 100; i++ {
			line := fmt.Sprintf("Block %d: %s\n", i+1, baseContent)
			_, err := fileWriter.Write([]byte(line))
			if err != nil {
				pw.CloseWithError(fmt.Errorf("error writing block %d: %w", i+1, err))
				return
			}

			// Simulate data processing
			if i%20 == 0 {
				fmt.Printf("Blocks written: %d/100\n", i+1)
			}
		}

		fmt.Println("All data written to multipart stream")
	}()

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s\n", resp.Status)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Response size: %d bytes\n", len(body))

	// Show part of response with file information
	bodyStr := string(body)
	if idx := strings.Index(bodyStr, `"large_file"`); idx != -1 {
		start := idx
		end := start + 200
		if end > len(bodyStr) {
			end = len(bodyStr)
		}
		fmt.Printf("Information about uploaded file:\n%s...\n", bodyStr[start:end])
	}

	fmt.Println("Streaming multipart request completed successfully!")
}
