# Concurrent Error Demonstration Package

This package demonstrates the critical errors that occur when multiple goroutines attempt to write concurrently to `io.Pipe` with multipart data.

## What This Package Shows

### 1. Main Demo (`multipart.go`)

Demonstrates the fundamental problem:
- ✅ **Correct Usage**: Sequential writing in a single goroutine
- ❌ **Incorrect Usage**: Multiple goroutines writing concurrently

**Key Learning**: While concurrent writes may appear to "work" (no immediate panic), they violate the multipart protocol and can corrupt data structure.

### 2. Advanced Demo (`boundary_demo/main.go`)

Shows deeper analysis of the corruption:
- Displays correct multipart structure
- Demonstrates race conditions leading to deadlocks
- Analyzes boundary corruption in multipart data

**Key Learning**: Concurrent writes to multipart writers cause deadlocks and data corruption.

## Why This Happens

According to the Go instructions for I/O and multipart handling:

> **Warning:** When using `io.Pipe` (especially with multipart writers), all writes must be performed in strict, sequential order. Do not write concurrently or out of order—multipart boundaries and chunk order must be preserved. Out-of-order or parallel writes can corrupt the stream and result in errors.

### Technical Reasons:

1. **Multipart Protocol Requirements**: Multipart data has strict boundary formatting that must be written sequentially
2. **Thread Safety**: `multipart.Writer` is not thread-safe and doesn't protect against concurrent access
3. **io.Pipe Behavior**: While `io.Pipe` itself is thread-safe, the multipart writer creates complex boundary structures that get corrupted when written concurrently
4. **Boundary Corruption**: Each field requires specific boundary markers written in exact order

## Running the Demonstrations

### Basic Error Demo:
```bash
go run http/request/concurrent_error/multipart.go
```

### Advanced Boundary Analysis:
```bash
go run http/request/concurrent_error/boundary_demo/main.go
```

## Expected Results

### Basic Demo:
- ✅ First part shows successful sequential writing
- ⚠️ Second part shows apparent "success" but with corrupted data

### Advanced Demo:
- ✅ Shows correct multipart structure
- 💥 **Deadlock occurs** - this is the expected behavior demonstrating the problem

## Best Practices Demonstrated

### ✅ DO:
```go
go func() {
    defer pw.Close()
    defer mw.Close()
    
    // Write ALL multipart data sequentially in this single goroutine
    mw.WriteField("field1", "value1")
    mw.WriteField("field2", "value2")
    
    fileWriter, _ := mw.CreateFormFile("file", "name.txt")
    fileWriter.Write(fileContent)
}()
```

### ❌ DON'T:
```go
// Multiple goroutines writing concurrently - THIS WILL FAIL
go func() {
    mw.WriteField("field1", "value1") // Goroutine 1
}()
go func() {
    mw.WriteField("field2", "value2") // Goroutine 2 - RACE CONDITION!
}()
```

## Key Takeaways

1. **Single Writer Rule**: Only one goroutine should write to a multipart writer
2. **Sequential Order**: All multipart parts must be written in sequence
3. **Deadlock Prevention**: Concurrent writes lead to deadlocks and data corruption
4. **Protocol Compliance**: Multipart format requires strict boundary ordering

This package serves as a practical demonstration of why the Go instructions emphasize sequential writing for multipart data with `io.Pipe`.