package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"latencyops/internal/domain"
	"latencyops/internal/service"
)

// Unit tests for worker pool job distribution and channel handling.
func TestWorkerPool_JobDistribution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// FIX: Use NewTestProbeService to bypass SSRF logic during unit test
	dummyValidator := func(url string) error { return nil }
	ps := service.NewTestProbeService(2*time.Second, dummyValidator)
	
	resultsChan := make(chan domain.PingResult, 10)
	pool := service.NewWorkerPool(3, ps, resultsChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	endpoint := domain.Endpoint{
		ID:        "test-uuid-001",
		TargetURL: server.URL,
	}
	pool.Dispatch(endpoint)

	select {
	case res := <-resultsChan:
		if !res.IsUp {
			t.Errorf("Expected endpoint to be up, but got down")
		}
		if res.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", res.StatusCode)
		}
		if res.EndpointID != "test-uuid-001" {
			t.Errorf("Expected Endpoint ID mapping to be accurate")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timeout waiting for worker result in results channel")
	}

	pool.Stop()
}

// Unit tests for timeout handling.
func TestProbeService_TimeoutHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	// FIX: Use NewTestProbeService to bypass SSRF logic during unit test
	dummyValidator := func(url string) error { return nil }
	ps := service.NewTestProbeService(10*time.Millisecond, dummyValidator)
	
	ctx := context.Background()
	endpoint := domain.Endpoint{
		ID:        "test-timeout-uuid",
		TargetURL: server.URL,
	}

	res, err := ps.ExecuteProbe(ctx, endpoint)
	
	if err == nil {
		t.Fatalf("Expected timeout error, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to execute ping probe") {
		t.Errorf("Expected explicit error wrapping, got: %v", err)
	}
	if res.IsUp {
		t.Errorf("Expected endpoint to be registered as down on timeout")
	}
}