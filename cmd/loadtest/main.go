package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"sync/atomic"
	"time"

	"latencyops/internal/domain"
	"latencyops/internal/service"
)

// loadtest is a custom load generator that stress-tests the LatencyOps Worker Engine
// pipeline end-to-end through local mock servers.
//
// Usage:
//   go run ./cmd/loadtest/main.go
//
// It spins up httptest servers with configurable latency jitter, dispatches them
// through the full WorkerPool pipeline, and reports resume-ready metrics.

func main() {
	log.Println("═══════════════════════════════════════════════")
	log.Println("  LatencyOps Load Test — Worker Engine Stress")
	log.Println("═══════════════════════════════════════════════")

	scenarios := []struct {
		name        string
		endpoints   int
		concurrency int
		serverDelay time.Duration
	}{
		{"Instant Response", 10_000, 100, 0},
		{"1ms Realistic API", 5_000, 50, 1 * time.Millisecond},
		{"5ms Slow API", 2_000, 50, 5 * time.Millisecond},
		{"High Concurrency Burst", 10_000, 200, 500 * time.Microsecond},
	}

	for _, sc := range scenarios {
		runScenario(sc.name, sc.endpoints, sc.concurrency, sc.serverDelay)
	}
}

func runScenario(name string, totalEndpoints, concurrency int, serverDelay time.Duration) {
	log.Printf("\n━━━ Scenario: %s ━━━", name)
	log.Printf("    Endpoints: %d | Concurrency: %d | Server Delay: %v\n", totalEndpoints, concurrency, serverDelay)

	// Spin up local mock server with configurable latency
	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if serverDelay > 0 {
			time.Sleep(serverDelay)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create the worker pipeline (bypassing SSRF for load testing)
	dummyValidator := func(url string) error { return nil }
	ps := service.NewTestProbeService(10*time.Second, dummyValidator)
	resultsChan := make(chan domain.PingResult, totalEndpoints+100)
	pool := service.NewWorkerPool(concurrency, ps, resultsChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	// Collect latencies for percentile analysis
	latencies := make([]time.Duration, 0, totalEndpoints)

	// Start timing
	start := time.Now()

	// Dispatch all endpoints
	go func() {
		for i := 0; i < totalEndpoints; i++ {
			pool.Dispatch(domain.Endpoint{
				ID:          fmt.Sprintf("load-ep-%d", i),
				WorkspaceID: "loadtest-workspace",
				TargetURL:   server.URL,
				Name:        fmt.Sprintf("Load Test Endpoint %d", i),
			})
		}
	}()

	// Collect all results
	successCount := 0
	failCount := 0
	for i := 0; i < totalEndpoints; i++ {
		result := <-resultsChan
		latencies = append(latencies, result.Latency)
		if result.IsUp {
			successCount++
		} else {
			failCount++
		}
	}

	totalDuration := time.Since(start)

	// Stop the pool
	pool.Stop()

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)
	minLat := latencies[0]
	maxLat := latencies[len(latencies)-1]

	throughput := float64(totalEndpoints) / totalDuration.Seconds()
	engineOverhead := totalDuration - time.Duration(totalEndpoints)*serverDelay/time.Duration(concurrency)

	// Memory stats
	var memStats [2]uint64
	memStats[0] = 0 // Simplified — use runtime.ReadMemStats for production profiling

	// Print Results
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Printf("│  %-55s │\n", name+" Results")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Printf("│  Total Endpoints:     %-33d │\n", totalEndpoints)
	fmt.Printf("│  Concurrency:         %-33d │\n", concurrency)
	fmt.Printf("│  Total Duration:      %-33s │\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("│  Throughput:          %-33s │\n", fmt.Sprintf("%.0f probes/sec", throughput))
	fmt.Printf("│  Success Rate:        %-33s │\n", fmt.Sprintf("%.1f%% (%d/%d)", float64(successCount)/float64(totalEndpoints)*100, successCount, totalEndpoints))
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Printf("│  P50 Latency:         %-33s │\n", p50.Round(time.Microsecond))
	fmt.Printf("│  P95 Latency:         %-33s │\n", p95.Round(time.Microsecond))
	fmt.Printf("│  P99 Latency:         %-33s │\n", p99.Round(time.Microsecond))
	fmt.Printf("│  Min Latency:         %-33s │\n", minLat.Round(time.Microsecond))
	fmt.Printf("│  Max Latency:         %-33s │\n", maxLat.Round(time.Microsecond))
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	if engineOverhead > 0 {
		fmt.Printf("│  Engine Overhead:     %-33s │\n", engineOverhead.Round(time.Millisecond))
	}
	fmt.Printf("│  Server Requests:     %-33d │\n", requestCount.Load())
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Resume-ready one-liner
	if totalEndpoints >= 10_000 && throughput > 1000 {
		fmt.Printf("📊 RESUME METRIC: \"Processed %s concurrent health checks at %.0f probes/sec with %s engine overhead\"\n",
			formatNumber(totalEndpoints), throughput, engineOverhead.Round(time.Millisecond))
	}
	if float64(successCount)/float64(totalEndpoints) >= 0.999 {
		fmt.Printf("📊 RESUME METRIC: \"Maintained %.1f%% success rate across %s probes\"\n",
			float64(successCount)/float64(totalEndpoints)*100, formatNumber(totalEndpoints))
	}

	// Prevent test server port exhaustion
	if _, ok := os.LookupEnv("CI"); !ok {
		time.Sleep(100 * time.Millisecond)
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func formatNumber(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%dK", n/1_000)
	}
	return fmt.Sprintf("%d", n)
}
