package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== Advanced Demonstration: Multipart Boundary Corruption ===")
	fmt.Println()

	fmt.Println("1. First, let's see what CORRECT multipart data looks like:")
	showCorrectMultipartStructure()

	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	fmt.Println("2. Now let's see what happens with RACE CONDITIONS:")
	demonstrateRaceCondition()

	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	fmt.Println("3. Finally, let's see CORRUPTED multipart boundaries:")
	demonstrateBoundaryCorruption()
}

// showCorrectMultipartStructure demonstrates what proper multipart data looks like
func showCorrectMultipartStructure() {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write fields in correct order
	writer.WriteField("field1", "value1")
	writer.WriteField("field2", "value2")

	fileWriter, _ := writer.CreateFormFile("file", "test.txt")
	fileWriter.Write([]byte("Test file content"))

	writer.Close()

	fmt.Printf("Content-Type: %s\n", writer.FormDataContentType())
	fmt.Printf("Correct multipart structure:\n%s\n", buf.String())
}

// demonstrateRaceCondition shows race conditions when multiple goroutines write
func demonstrateRaceCondition() {
	fmt.Println("Creating io.Pipe with concurrent writers...")

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// Capture the multipart data to analyze
	var capturedData bytes.Buffer
	capturedReader := io.TeeReader(pr, &capturedData)

	// Create request
	req, _ := http.NewRequest("POST", "https://httpbin.org/post", capturedReader)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// PROBLEMATIC: Multiple goroutines writing concurrently
	// This violates multipart protocol requirements
	// Note: Using WaitGroup.Go() method (Go 1.25+) instead of Add/Done pattern

	// Writer 1: Tries to write immediately
	wg.Go(func() {
		err := mw.WriteField("concurrent_field1", "This field might get corrupted")
		if err != nil {
			errChan <- fmt.Errorf("writer 1 error: %w", err)
		} else {
			fmt.Println("[WARNING] Writer 1 completed (timing: immediate)")
		}
	})

	// Writer 2: Tries to write with small delay
	wg.Go(func() {
		time.Sleep(1 * time.Millisecond)
		err := mw.WriteField("concurrent_field2", "This field might also get corrupted")
		if err != nil {
			errChan <- fmt.Errorf("writer 2 error: %w", err)
		} else {
			fmt.Println("[WARNING] Writer 2 completed (timing: 1ms delay)")
		}
	})

	// Writer 3: Tries to create file field
	wg.Go(func() {
		time.Sleep(2 * time.Millisecond)
		fileWriter, err := mw.CreateFormFile("concurrent_file", "racing.txt")
		if err != nil {
			errChan <- fmt.Errorf("writer 3 create error: %w", err)
			return
		}
		_, err = fileWriter.Write([]byte("File content written during race condition"))
		if err != nil {
			errChan <- fmt.Errorf("writer 3 write error: %w", err)
		} else {
			fmt.Println("[WARNING] Writer 3 completed (timing: 2ms delay)")
		}
	})

	// Close pipe after all writers complete
	go func() {
		wg.Wait()
		close(errChan)
		mw.Close()
		pw.Close()
	}()

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		fmt.Printf("[ERROR] Errors encountered during concurrent writing:\n")
		for i, err := range errors {
			fmt.Printf("  %d. %v\n", i+1, err)
		}
	}

	// Try to send the request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Request failed: %v\n", err)
		fmt.Println("   This is expected due to corrupted multipart data")
	} else {
		defer resp.Body.Close()
		fmt.Printf("[WARNING] Request succeeded with status: %s\n", resp.Status)
		fmt.Println("   But the multipart data structure may be corrupted")
	}

	// Show the captured multipart data structure
	captured := capturedData.String()
	if len(captured) > 0 {
		fmt.Printf("\nCaptured multipart data (first 500 chars):\n%s\n",
			captured[:min(500, len(captured))])

		// Analyze the structure
		boundaryCount := strings.Count(captured, "--")
		fmt.Printf("Analysis: Found %d boundary markers\n", boundaryCount)

		if strings.Contains(captured, "concurrent_field1") &&
			strings.Contains(captured, "concurrent_field2") {
			fmt.Println("[WARNING] Both fields present, but order and structure may be corrupted")
		}
	}
}

// demonstrateBoundaryCorruption shows how concurrent writes corrupt multipart boundaries
func demonstrateBoundaryCorruption() {
	fmt.Println("Demonstrating boundary corruption with intentional timing conflicts...")

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// Buffer to capture corrupted output
	var corruptedBuffer bytes.Buffer
	teeReader := io.TeeReader(pr, &corruptedBuffer)

	// Start reading in background
	go func() {
		io.Copy(io.Discard, teeReader) // Just consume the data
	}()

	var wg sync.WaitGroup

	// Create intentional timing conflicts to corrupt boundaries
	// Using WaitGroup.Go() for cleaner concurrent task management
	for i := 0; i < 5; i++ {
		wg.Go(func() {
			index := i
			// Each goroutine waits a different amount and tries to write
			time.Sleep(time.Duration(index) * time.Millisecond)

			fieldName := fmt.Sprintf("racing_field_%d", index)
			fieldValue := fmt.Sprintf("Value written by goroutine %d at time %v",
				index, time.Now().UnixNano())

			err := mw.WriteField(fieldName, fieldValue)
			if err != nil {
				fmt.Printf("[ERROR] Goroutine %d failed: %v\n", index, err)
			} else {
				fmt.Printf("[WARNING] Goroutine %d wrote field (may be corrupted)\n", index)
			}
		})
	}

	// Close after all goroutines finish
	go func() {
		wg.Wait()
		mw.Close()
		pw.Close()
	}()

	// Wait for data to be captured
	time.Sleep(100 * time.Millisecond)

	// Analyze the corruption
	corrupted := corruptedBuffer.String()
	fmt.Printf("\nCorrupted multipart data analysis:\n")
	fmt.Printf("Total size: %d bytes\n", len(corrupted))

	lines := strings.Split(corrupted, "\n")
	fmt.Printf("Number of lines: %d\n", len(lines))

	boundaryLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "--") {
			boundaryLines++
		}
	}
	fmt.Printf("Boundary lines found: %d\n", boundaryLines)

	if boundaryLines != 6 { // Should be 5 fields + 1 closing boundary
		fmt.Printf("[ERROR] CORRUPTION DETECTED: Expected 6 boundary lines, found %d\n", boundaryLines)
		fmt.Println("  This indicates the multipart structure is corrupted!")
	}

	// Show a sample of the corrupted data
	if len(corrupted) > 0 {
		sample := corrupted
		if len(sample) > 800 {
			sample = sample[:800] + "..."
		}
		fmt.Printf("\nSample of corrupted data:\n%s\n", sample)
	}

	fmt.Println("\n[CRITICAL] CONCLUSION:")
	fmt.Println("   Concurrent writes to io.Pipe with multipart data WILL corrupt")
	fmt.Println("   the multipart boundaries and field structure, making the data")
	fmt.Println("   unparseable by HTTP servers and clients!")
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
