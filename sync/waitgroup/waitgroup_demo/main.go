package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== WaitGroup.Go() Method Demo (Go 1.25+) ===")
	fmt.Println()

	demonstrateNewWaitGroupAPI()
}

// demonstrateNewWaitGroupAPI shows the new WaitGroup.Go() method usage
func demonstrateNewWaitGroupAPI() {
	fmt.Println("Demonstrating WaitGroup.Go() method (introduced in Go 1.25):")
	fmt.Println()

	var wg sync.WaitGroup

	// Old way (still works but more verbose):
	// wg.Add(1)
	// go func() {
	//     defer wg.Done()
	//     // task code here
	// }()

	// New way using WaitGroup.Go() method:
	wg.Go(func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("[Task 1] Completed work in goroutine 1")
	})

	wg.Go(func() {
		time.Sleep(200 * time.Millisecond)
		fmt.Println("[Task 2] Completed work in goroutine 2")
	})

	wg.Go(func() {
		time.Sleep(50 * time.Millisecond)
		fmt.Println("[Task 3] Completed work in goroutine 3")
	})

	fmt.Println("Waiting for all goroutines to complete...")
	wg.Wait()
	fmt.Println("All tasks completed!")

	fmt.Println()
	fmt.Println("Benefits of WaitGroup.Go():")
	fmt.Println("- Cleaner syntax: no need for Add/Done pattern")
	fmt.Println("- Less error-prone: no risk of forgetting defer wg.Done()")
	fmt.Println("- More readable: task definition is clearer")
}
