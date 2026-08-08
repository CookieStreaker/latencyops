package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"latencyops/internal/domain"
)

type ProbeService interface {
	ExecuteProbe(ctx context.Context, endpoint domain.Endpoint) (domain.PingResult, error)
}

type probeService struct {
	client    *http.Client
	validator func(string) error // Dependency Injection for SSRF Validation
}

// NewProbeService initializes a strict HTTP client with timeouts for Production.
func NewProbeService(timeout time.Duration) ProbeService {
	return &probeService{
		client: &http.Client{
			Timeout: timeout,
		},
		validator: domain.ValidateURL, // Default to our strict OWASP validation
	}
}

// NewTestProbeService allows bypassing SSRF checks strictly for internal unit testing.
func NewTestProbeService(timeout time.Duration, mockValidator func(string) error) ProbeService {
	return &probeService{
		client: &http.Client{Timeout: timeout},
		validator: mockValidator,
	}
}

func (s *probeService) ExecuteProbe(ctx context.Context, endpoint domain.Endpoint) (domain.PingResult, error) {
	// 1. Re-validate URL using the injected validator
	if err := s.validator(endpoint.TargetURL); err != nil {
		return domain.PingResult{}, fmt.Errorf("probe validation failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.TargetURL, nil)
	if err != nil {
		return domain.PingResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := s.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return domain.PingResult{
			EndpointID: endpoint.ID,
			StatusCode: 0,
			Latency:    latency,
			IsUp:       false,
			Timestamp:  time.Now(),
		}, fmt.Errorf("failed to execute ping probe: %w", err) 
	}
	defer resp.Body.Close()

	isUp := resp.StatusCode >= 200 && resp.StatusCode < 300

	return domain.PingResult{
		EndpointID: endpoint.ID,
		StatusCode: resp.StatusCode,
		Latency:    latency,
		IsUp:       isUp,
		Timestamp:  time.Now(),
	}, nil
}