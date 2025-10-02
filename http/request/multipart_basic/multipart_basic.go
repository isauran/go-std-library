package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func main() {
	fmt.Println("=== Multipart HTTP Request Demo ===")
	fmt.Println()

	// Example 1: Creating multipart form with text fields
	fmt.Println("1. Creating multipart form with text fields:")
	createTextFieldsExample()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// Example 2: Creating multipart form with file upload
	fmt.Println("2. Creating multipart form with file upload:")
	createFileUploadExample()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// Example 3: Complete example of sending multipart request
	fmt.Println("3. Complete example of sending multipart request:")
	sendMultipartRequestExample()
}

// createTextFieldsExample demonstrates creating a multipart form with text fields
func createTextFieldsExample() {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add text fields
	writer.WriteField("username", "john_doe")
	writer.WriteField("email", "john@example.com")
	writer.WriteField("age", "25")

	// Закрываем writer
	writer.Close()

	fmt.Printf("Content-Type: %s\n", writer.FormDataContentType())
	fmt.Printf("Data size: %d bytes\n", buf.Len())
	fmt.Printf("First 200 characters:\n%s...\n", buf.String()[:min(200, buf.Len())])
}

// createFileUploadExample demonstrates creating a multipart form with file upload
func createFileUploadExample() {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add text field
	writer.WriteField("description", "Example file")

	// Create file field
	fileWriter, err := writer.CreateFormFile("upload", "example.txt")
	if err != nil {
		fmt.Printf("Error creating file field: %v\n", err)
		return
	}

	// Write file content
	fileContent := "This is example file content\nSecond line of file\n"
	_, err = fileWriter.Write([]byte(fileContent))
	if err != nil {
		fmt.Printf("Ошибка записи файла: %v\n", err)
		return
	}

	// Закрываем writer
	writer.Close()

	fmt.Printf("Content-Type: %s\n", writer.FormDataContentType())
	fmt.Printf("Data size: %d bytes\n", buf.Len())
	fmt.Printf("First 300 characters:\n%s...\n", buf.String()[:min(300, buf.Len())])
}

// sendMultipartRequestExample demonstrates complete cycle of creating and sending multipart request
func sendMultipartRequestExample() {
	// Create buffer for multipart data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	writer.WriteField("action", "upload")
	writer.WriteField("user_id", "12345")

	// Add file
	fileWriter, err := writer.CreateFormFile("document", "report.txt")
	if err != nil {
		fmt.Printf("Error creating file field: %v\n", err)
		return
	}

	reportContent := `System Status Report
Date: 01.10.2025
Status: Success
Details: All components are working normally`

	_, err = fileWriter.Write([]byte(reportContent))
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	// Close the writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://httpbin.org/post", &buf)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Set proper Content-Type
	req.Header.Set("Content-Type", contentType)

	fmt.Printf("Method: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	fmt.Printf("Content-Type: %s\n", req.Header.Get("Content-Type"))
	fmt.Printf("Request body size: %d bytes\n", buf.Len())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s\n", resp.Status)

	// Read and display part of response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Response size: %d bytes\n", len(body))
	if len(body) > 500 {
		fmt.Printf("First 500 characters of response:\n%s...\n", string(body[:500]))
	} else {
		fmt.Printf("Response:\n%s\n", string(body))
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
