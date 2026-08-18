package service_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"latencyops/internal/domain"
	"latencyops/internal/service"
)

// BenchmarkWorkerPool_Throughput measures end-to-end throughput of the worker pool
// (dispatch → HTTP probe → result collection) using a local httptest server.
// This produces the "Processed N health checks per second" resume metric.
func BenchmarkWorkerPool_Throughput(b *testing.B) {
	// Local server that returns instantly (isolates engine overhead from network latency)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dummyValidator := func(url string) error { return nil }
	ps := service.NewTestProbeService(5*time.Second, dummyValidator)

	concurrency := 50
	resultsChan := make(chan domain.PingResult, b.N+1000)
	pool := service.NewWorkerPool(concurrency, ps, resultsChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	// Dispatch all jobs
	for i := 0; i < b.N; i++ {
		pool.Dispatch(domain.Endpoint{
			ID:          fmt.Sprintf("bench-ep-%d", i),
			WorkspaceID: "bench-workspace",
			TargetURL:   server.URL,
		})
	}

	// Drain all results
	for i := 0; i < b.N; i++ {
		<-resultsChan
	}

	b.StopTimer()
	pool.Stop()
}

// BenchmarkProbeService_Latency isolates the HTTP probe execution overhead.
// Measures per-probe cost (allocs, ns/op) without worker pool coordination.
func BenchmarkProbeService_Latency(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dummyValidator := func(url string) error { return nil }
	ps := service.NewTestProbeService(5*time.Second, dummyValidator)
	ctx := context.Background()

	endpoint := domain.Endpoint{
		ID:          "bench-probe-ep",
		WorkspaceID: "bench-workspace",
		TargetURL:   server.URL,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ps.ExecuteProbe(ctx, endpoint)
		if err != nil {
			b.Fatalf("probe failed: %v", err)
		}
	}
}

// BenchmarkWorkerPool_ScaledConcurrency tests pool throughput at different concurrency levels.
// This helps find the optimal worker count and produces data for
// "Throughput scaling: 10→50→100→200 workers" analysis.
func BenchmarkWorkerPool_ScaledConcurrency(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate realistic 1ms processing delay
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dummyValidator := func(url string) error { return nil }

	concurrencyLevels := []int{10, 50, 100, 200}

	for _, c := range concurrencyLevels {
		b.Run(fmt.Sprintf("Workers_%d", c), func(b *testing.B) {
			ps := service.NewTestProbeService(5*time.Second, dummyValidator)
			resultsChan := make(chan domain.PingResult, b.N+1000)
			pool := service.NewWorkerPool(c, ps, resultsChan)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pool.Start(ctx)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pool.Dispatch(domain.Endpoint{
					ID:          fmt.Sprintf("scaled-ep-%d", i),
					WorkspaceID: "bench-workspace",
					TargetURL:   server.URL,
				})
			}

			for i := 0; i < b.N; i++ {
				<-resultsChan
			}

			b.StopTimer()
			pool.Stop()
		})
	}
}

// BenchmarkWorkerPool_BurstCapacity simulates a burst of 10,000 endpoints
// dispatched simultaneously and measures total processing time.
// Generates the "Processed 10,000 concurrent health checks with <Xms engine overhead" metric.
func BenchmarkWorkerPool_BurstCapacity(b *testing.B) {
	var requestCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dummyValidator := func(url string) error { return nil }
	ps := service.NewTestProbeService(5*time.Second, dummyValidator)

	const burstSize = 10_000
	const concurrency = 100

	resultsChan := make(chan domain.PingResult, burstSize+100)
	pool := service.NewWorkerPool(concurrency, ps, resultsChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		requestCount.Store(0)
		start := time.Now()

		// Dispatch burst
		for j := 0; j < burstSize; j++ {
			pool.Dispatch(domain.Endpoint{
				ID:          fmt.Sprintf("burst-ep-%d", j),
				WorkspaceID: "bench-workspace",
				TargetURL:   server.URL,
			})
		}

		// Drain all results
		for j := 0; j < burstSize; j++ {
			<-resultsChan
		}

		elapsed := time.Since(start)
		processed := requestCount.Load()
		b.ReportMetric(float64(processed)/elapsed.Seconds(), "probes/sec")
		b.ReportMetric(float64(elapsed.Milliseconds()), "burst_ms")
	}

	b.StopTimer()
	pool.Stop()
}
