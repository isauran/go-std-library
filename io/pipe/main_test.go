package main

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestBuilder(t *testing.T) {
	builder, err := NewBuilder()
	if err != nil {
		t.Fatal("Error creating builder:", err)
	}
	stats := builder.
		String("test1").
		String("test2").
		JSON(map[string]string{"key": "value"}).
		Build()

	if stats["string"] != 2 {
		t.Errorf("Expected 2 strings, got %d", stats["string"])
	}
	if stats["json"] != 1 {
		t.Errorf("Expected 1 json, got %d", stats["json"])
	}

	// Check file exists
	if _, err := os.Stat("output.multipart"); os.IsNotExist(err) {
		t.Error("output.multipart not created")
	}

	// Check file has content
	file, err := os.Open("output.multipart")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	content := ""
	for scanner.Scan() {
		content += scanner.Text() + "\n"
	}
	if len(content) == 0 {
		t.Error("File is empty")
	}
	if !strings.Contains(content, "test1") || !strings.Contains(content, `"key":"value"`) {
		t.Error("File does not contain expected content")
	}
}

func BenchmarkBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builder, _ := NewBuilder()
		builder.
			String("line").
			JSON(map[string]int{"num": i}).
			Build()
		os.Remove("output.multipart") // Clean up
	}
}
