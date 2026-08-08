package service

import (
	"context"
	"log"
	"sync"

	"latencyops/internal/domain"
)

// WorkerPool implements bounded concurrency logic using Go channels.
type WorkerPool struct {
	concurrency  int
	probeService ProbeService
	jobs         chan domain.Endpoint
	results      chan domain.PingResult
	wg           sync.WaitGroup
}

// NewWorkerPool initializes the pool to prevent goroutine leaks when pinging thousands of endpoints[cite: 76].
func NewWorkerPool(concurrency int, ps ProbeService, resultsChan chan domain.PingResult) *WorkerPool {
	return &WorkerPool{
		concurrency:  concurrency,
		probeService: ps,
		jobs:         make(chan domain.Endpoint, concurrency*2), // Buffered channel to handle sudden bursts
		results:      resultsChan,
	}
}

// Start spins up the goroutine worker pool[cite: 31].
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.concurrency; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

// worker represents a single goroutine inside the pool executing HTTP requests[cite: 68].
func (wp *WorkerPool) worker(ctx context.Context, workerID int) {
	defer wp.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, exit gracefully
		case endpoint, ok := <-wp.jobs:
			if !ok {
				return // Job channel closed
			}
			
			// Execute the probe logic [cite: 36]
			result, err := wp.probeService.ExecuteProbe(ctx, endpoint)
			if err != nil {
				// We log the error but still pass the degraded state to Redis/DB down the line [cite: 34]
				log.Printf("[Worker %d] Error probing %s: %v", workerID, endpoint.TargetURL, err)
			}
			
			// Forward ping outputs to the results channel [cite: 34]
			select {
			case wp.results <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Dispatch pushes jobs to the bounded Go channel[cite: 31, 56].
func (wp *WorkerPool) Dispatch(endpoint domain.Endpoint) {
	wp.jobs <- endpoint
}

// Stop gracefully shuts down the worker pool, preventing leaks[cite: 76].
func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
}