### 1. WaitGroup.Go() Method Demo (`waitgroup_demo/`)

Demonstrates the new `WaitGroup.Go()` method introduced in Go 1.25:

- **Modern API Usage**: Shows the new `WaitGroup.Go()` method
- **Cleaner Syntax**: Eliminates need for Add/Done pattern  
- **Error Prevention**: Reduces risk of forgetting `defer wg.Done()`
- **Better Readability**: Clearer task definition

#### Features Demonstrated:
- New `WaitGroup.Go()` method syntax
- Comparison with old Add/Done pattern
- Benefits of the new API
- Proper concurrent task management

#### Usage:
```bash
go run waitgroup_demo/main.go
```
## Requirements

- Go 1.25.1 or later