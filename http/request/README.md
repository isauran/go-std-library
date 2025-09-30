# Multipart HTTP Request Examples

This directory contains comprehensive examples demonstrating multipart HTTP requests in Go using the standard library.

## Examples Included

### 1. Basic Multipart Form (`multipart_basic/multipart.go`)

Demonstrates three different approaches to creating and sending multipart HTTP requests:

- **Text Fields Only**: Creating a multipart form with simple text fields
- **File Upload**: Creating a multipart form with file upload capabilities  
- **Complete Request**: Full example showing the entire process of creating and sending a multipart request

#### Features Demonstrated:
- Using `mime/multipart` package for form creation
- Setting proper Content-Type headers
- Adding text fields with `WriteField()`
- Adding file uploads with `CreateFormFile()`
- Sending requests to a test endpoint
- Reading and displaying response data

#### Usage:
```bash
go run multipart_basic/multipart.go
```

### 2. Streaming Multipart (`multipart_streaming/multipart.go`)

Shows how to use streaming multipart for large files using `io.Pipe`:

- **Memory Efficient**: Streams data without buffering everything in memory
- **Concurrent Processing**: Uses goroutines for concurrent writing
- **Large File Simulation**: Demonstrates handling large files by writing in blocks
- **Error Handling**: Proper error propagation through the pipe

#### Features Demonstrated:
- Using `io.Pipe` for streaming multipart data
- Concurrent writing with goroutines
- Proper cleanup with `defer` statements
- Error handling with `CloseWithError()`
- Progress tracking during upload

#### Usage:
```bash
go run multipart_streaming/multipart.go
```

### 3. Concurrent Error Demonstration (`concurrent_error/`)

🚨 **CRITICAL**: Demonstrates what happens when multiple goroutines incorrectly write to `io.Pipe` concurrently:

- **Correct Sequential Writing**: Shows the proper way to write multipart data
- **Concurrent Writing Errors**: Demonstrates race conditions and deadlocks
- **Boundary Corruption Analysis**: Shows how concurrent writes corrupt multipart boundaries
- **Educational Deadlocks**: Intentionally triggers deadlocks to show the problem

#### Features Demonstrated:
- Why concurrent writes to `multipart.Writer` fail
- Race conditions in multipart boundary writing
- Deadlock scenarios with concurrent goroutines
- Proper vs improper use of `io.Pipe` with multipart data
- Analysis of corrupted multipart structure

#### Usage:
```bash
# Basic concurrent error demo
go run concurrent_error/multipart.go

# Advanced boundary corruption analysis (will deadlock - this is expected)
go run concurrent_error/boundary_demo/main.go
```

**⚠️ Warning**: The advanced demo intentionally causes deadlocks to demonstrate the problem. This is expected behavior showing why concurrent writes must be avoided.

## Key Go Standard Library Packages Used

- **`mime/multipart`**: Core package for creating multipart forms
- **`net/http`**: HTTP client and request handling
- **`io`**: I/O operations including pipes for streaming
- **`bytes`**: Byte buffer operations for non-streaming examples

## Best Practices Demonstrated

1. **Proper Resource Management**: All writers and readers are properly closed
2. **Error Handling**: Comprehensive error checking at each step
3. **Memory Efficiency**: Streaming approach for large files
4. **Concurrent Safety**: Proper use of goroutines and synchronization
5. **Idiomatic Go**: Following Go best practices and conventions
6. **🚨 Sequential Writing Rule**: Critical demonstration of why multipart data must be written sequentially, not concurrently
7. **Error Prevention**: Shows common pitfalls and how to avoid them

## Testing

Both examples use `httpbin.org/post` as a test endpoint which echoes back the received data, making it easy to verify that the multipart data was sent correctly.

## Requirements

- Go 1.25.1 or later
- Internet connection (for sending requests to httpbin.org)

## Notes

- The examples include comprehensive error handling
- All comments follow Go documentation standards
- Code follows idiomatic Go practices as outlined in Effective Go
- No external dependencies beyond the Go standard library