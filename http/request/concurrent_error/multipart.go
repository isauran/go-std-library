package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== Demonstration of io.Pipe Concurrent Write Error ===")
	fmt.Println()

	fmt.Println("1. Showing CORRECT sequential multipart writing:")
	demonstrateCorrectUsage()

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	fmt.Println("2. Showing INCORRECT concurrent multipart writing (will cause errors):")
	demonstrateConcurrentError()
}

// demonstrateCorrectUsage shows the proper way to write multipart data sequentially
func demonstrateCorrectUsage() {
	pr, pw := io.Pipe()

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://httpbin.org/post", pr)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Create multipart writer
	mw := multipart.NewWriter(pw)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	fmt.Println("Starting CORRECT sequential writing...")

	// Single goroutine writes ALL data sequentially
	go func() {
		defer pw.Close()
		defer mw.Close()

		// Write field 1
		if err := mw.WriteField("field1", "value1"); err != nil {
			pw.CloseWithError(fmt.Errorf("error writing field1: %w", err))
			return
		}
		fmt.Println("[OK] Written field1 sequentially")

		// Write field 2
		if err := mw.WriteField("field2", "value2"); err != nil {
			pw.CloseWithError(fmt.Errorf("error writing field2: %w", err))
			return
		}
		fmt.Println("[OK] Written field2 sequentially")

		// Write file
		fileWriter, err := mw.CreateFormFile("file", "test.txt")
		if err != nil {
			pw.CloseWithError(fmt.Errorf("error creating file field: %w", err))
			return
		}

		_, err = fileWriter.Write([]byte("Sequential file content"))
		if err != nil {
			pw.CloseWithError(fmt.Errorf("error writing file: %w", err))
			return
		}
		fmt.Println("[OK] Written file sequentially")

		fmt.Println("[OK] All data written correctly in sequence")
	}()

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("[OK] Response status: %s\n", resp.Status)
}

// demonstrateConcurrentError shows what happens when multiple goroutines write concurrently
func demonstrateConcurrentError() {
	pr, pw := io.Pipe()

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://httpbin.org/post", pr)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Create multipart writer
	mw := multipart.NewWriter(pw)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	fmt.Println("Starting INCORRECT concurrent writing...")

	var wg sync.WaitGroup

	// WRONG: Multiple goroutines writing concurrently to the same multipart writer
	// This violates the rule that multipart boundaries must be written in strict order
	// Note: Using WaitGroup.Go() method (Go 1.25+) instead of Add/Done pattern

	// Goroutine 1: tries to write field1
	wg.Go(func() {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		if err := mw.WriteField("field1", "value1"); err != nil {
			fmt.Printf("[ERROR] Error in goroutine 1 writing field1: %v\n", err)
		} else {
			fmt.Println("[UNCERTAIN] Goroutine 1 wrote field1 (may be corrupted)")
		}
	})

	// Goroutine 2: tries to write field2 concurrently
	wg.Go(func() {
		time.Sleep(5 * time.Millisecond) // Different timing
		if err := mw.WriteField("field2", "value2"); err != nil {
			fmt.Printf("[ERROR] Error in goroutine 2 writing field2: %v\n", err)
		} else {
			fmt.Println("[UNCERTAIN] Goroutine 2 wrote field2 (may be corrupted)")
		}
	})

	// Goroutine 3: tries to create and write file concurrently
	wg.Go(func() {
		time.Sleep(15 * time.Millisecond) // Yet another timing

		fileWriter, err := mw.CreateFormFile("file", "test.txt")
		if err != nil {
			fmt.Printf("[ERROR] Error in goroutine 3 creating file field: %v\n", err)
			return
		}

		_, err = fileWriter.Write([]byte("Concurrent file content"))
		if err != nil {
			fmt.Printf("[ERROR] Error in goroutine 3 writing file: %v\n", err)
		} else {
			fmt.Println("[UNCERTAIN] Goroutine 3 wrote file (may be corrupted)")
		}
	})

	// Wait for all concurrent writes and then close
	go func() {
		wg.Wait()
		mw.Close()
		pw.Close()
		fmt.Println("[ERROR] Closed multipart writer after concurrent operations")
	}()

	// Try to send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Error sending request (expected due to corrupted multipart): %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Even if the request succeeds, the multipart data is likely corrupted
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[ERROR] Error reading response: %v\n", err)
		return
	}

	fmt.Printf("[ERROR] Response status: %s (but multipart data is likely corrupted)\n", resp.Status)
	fmt.Printf("[ERROR] Response size: %d bytes\n", len(body))

	// Check if the multipart data was properly parsed
	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr != "{}" {
		fmt.Println("[WARNING] Server received some data, but multipart structure may be corrupted")
		fmt.Println("[WARNING] This demonstrates why concurrent writes to io.Pipe with multipart are dangerous")
	}
}
