package repository

import (
	"context"
	"fmt"

	"latencyops/internal/domain"
)

// PingResultRepository defines the persistence operations for health check results.
type PingResultRepository interface {
	SavePingResult(ctx context.Context, result domain.PingResult) error
}

type postgresPingResultRepo struct {
	db PgxEngine
}

// NewPostgresPingResultRepo initializes the ping result repository.
func NewPostgresPingResultRepo(db PgxEngine) PingResultRepository {
	return &postgresPingResultRepo{db: db}
}

// SavePingResult persists a health check result using strictly parameterized queries.
// The workspace_id is enforced through the FK chain (ping_results.endpoint_id → endpoints.workspace_id).
func (r *postgresPingResultRepo) SavePingResult(ctx context.Context, result domain.PingResult) error {
	query := `
		INSERT INTO ping_results (endpoint_id, status_code, latency_ms, is_up, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		result.EndpointID,
		result.StatusCode,
		result.LatencyMs,
		result.IsUp,
		result.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert ping result for endpoint %s: %w", result.EndpointID, err)
	}

	return nil
}
