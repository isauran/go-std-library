package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"sync"
)

type Data struct {
	FileType string
	Value    any
}

type Builder struct {
	ch    chan Data
	wg    sync.WaitGroup
	mw    *multipart.Writer
	pr    *io.PipeReader
	pw    *io.PipeWriter
	stats map[string]int
}

func NewBuilder() (*Builder, error) {
	file, err := os.Create("output.multipart")
	if err != nil {
		return nil, err
	}
	pipeReader, pipeWriter := io.Pipe()
	ch := make(chan Data) // Unbuffered channel to preserve the order of operations.
	b := &Builder{
		ch:    ch,
		pr:    pipeReader,
		pw:    pipeWriter,
		stats: make(map[string]int),
		mw:    multipart.NewWriter(pipeWriter),
	}
	// Start copying in a goroutine.
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		io.Copy(file, b.pr)
	}()
	b.wg.Add(1)
	go b.worker()
	return b, nil
}

func (b *Builder) worker() {
	defer b.wg.Done()
	defer b.mw.Close()
	defer b.pw.Close()
	for data := range b.ch {
		if data.FileType == "string" {
			if str, ok := data.Value.(string); ok {
				err := b.mw.WriteField("string", str)
				if err != nil {
					fmt.Println("Error writing field:", err)
					continue
				}
			}
		} else if data.FileType == "json" {
			part, err := b.mw.CreateFormFile("json", "data.json")
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
		b.stats[data.FileType]++
	}
}

func (b *Builder) String(line string) *Builder {
	b.ch <- Data{FileType: "string", Value: line}
	return b
}

func (b *Builder) JSON(j any) *Builder {
	b.ch <- Data{FileType: "json", Value: j}
	return b
}

func (b *Builder) Build() map[string]int {
	close(b.ch)
	b.wg.Wait()
	return b.stats
}

func main() {
	builder, err := NewBuilder()
	if err != nil {
		fmt.Println("Error creating builder:", err)
		return
	}
	stats := builder.
		String("1").
		String("2").
		String("3").
		JSON(map[string]string{"key": "value"}).
		Build()
	fmt.Printf("stats: %v\n", stats)
}
