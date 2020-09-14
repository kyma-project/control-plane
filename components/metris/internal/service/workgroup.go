package service

import (
	"context"
	"sync"
)

// Workgroup manages the life of goroutines.
type Workgroup struct {
	fn []func(<-chan struct{}) error
}

// Add adds a function to the Workgroup.
func (g *Workgroup) Add(fn func(<-chan struct{}) error) {
	g.fn = append(g.fn, fn)
}

// AddWithContext adds a function with a context to the Workgroup.
func (g *Workgroup) AddWithContext(fn func(context.Context)) {
	g.fn = append(g.fn, func(stop <-chan struct{}) error {
		fnctx, fncancel := context.WithCancel(context.Background())
		done := make(chan int)

		go func() {
			defer close(done)
			fn(fnctx)
		}()

		// wait for stop signal from the workgroup
		<-stop

		// cancel the context passed to the function
		fncancel()

		// wait for function to exit
		<-done

		return nil
	})
}

// Run executes each function from the workgroup in its own goroutine.
func (g *Workgroup) Run() error {
	if len(g.fn) < 1 {
		return nil
	}

	var wg sync.WaitGroup

	wg.Add(len(g.fn))

	stop := make(chan struct{})
	result := make(chan error, len(g.fn))

	// start each function in their own goroutine
	for _, fn := range g.fn {
		go func(fn func(<-chan struct{}) error) {
			defer wg.Done()
			result <- fn(stop)
		}(fn)
	}

	// as soon as the first function return it will send the stop signal to all
	// functions and blocks until they have all returned
	defer wg.Wait()
	defer close(stop)

	// return the result of the first function that finish
	return <-result
}
