package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"latencyops/internal/domain"
	"latencyops/internal/repository"
)

func TestPostgresPingResultRepo_SavePingResult(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := repository.NewPostgresPingResultRepo(mock)
	now := time.Now()

	result := domain.PingResult{
		WorkspaceID: "workspace-uuid-001",
		EndpointID:  "endpoint-uuid-001",
		StatusCode:  200,
		Latency:     42 * time.Millisecond,
		LatencyMs:   42,
		IsUp:        true,
		Timestamp:   now,
	}

	// ASSERTION: Ensure INSERT uses all 5 parameterized arguments
	// Verify latency_ms is stored as integer, not time.Duration
	mock.ExpectExec(`INSERT INTO ping_results`).
		WithArgs(result.EndpointID, result.StatusCode, result.LatencyMs, result.IsUp, result.Timestamp).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := context.Background()
	err = repo.SavePingResult(ctx, result)

	if err != nil {
		t.Errorf("error was not expected while saving ping result: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestPostgresPingResultRepo_SavePingResult_DownEndpoint(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := repository.NewPostgresPingResultRepo(mock)
	now := time.Now()

	result := domain.PingResult{
		WorkspaceID: "workspace-uuid-002",
		EndpointID:  "endpoint-uuid-002",
		StatusCode:  503,
		Latency:     5 * time.Second,
		LatencyMs:   5000,
		IsUp:        false,
		Timestamp:   now,
	}

	mock.ExpectExec(`INSERT INTO ping_results`).
		WithArgs(result.EndpointID, result.StatusCode, result.LatencyMs, result.IsUp, result.Timestamp).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := context.Background()
	err = repo.SavePingResult(ctx, result)

	if err != nil {
		t.Errorf("error was not expected while saving down endpoint result: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
