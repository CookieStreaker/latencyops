package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"latencyops/internal/domain"
)

// PgxEngine defines the database interface needed by postgresEndpointRepo.
// Both *pgxpool.Pool (Production) and pgxmock.PgxPoolIface (Testing) implement this interface.
type PgxEngine interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// EndpointRepository defines the CRUD operations for Endpoints.
type EndpointRepository interface {
	GetByID(ctx context.Context, id, workspaceID string) (*domain.Endpoint, error)
	GetActiveEndpoints(ctx context.Context) ([]domain.Endpoint, error)
	Save(ctx context.Context, endpoint *domain.Endpoint) error
}

type postgresEndpointRepo struct {
	db PgxEngine
}

// NewPostgresEndpointRepo initializes the repository, accepting any valid PgxEngine.
func NewPostgresEndpointRepo(db PgxEngine) EndpointRepository {
	return &postgresEndpointRepo{db: db}
}

// GetByID enforces strict OWASP API1 BOLA mitigation by validating workspace_id on the query level.
func (r *postgresEndpointRepo) GetByID(ctx context.Context, id, workspaceID string) (*domain.Endpoint, error) {
	// EVERY query fetching resources MUST enforce ownership.
	query := `
		SELECT id, workspace_id, name, target_url, interval_seconds, created_at
		FROM endpoints
		WHERE id = $1 AND workspace_id = $2
	`
	row := r.db.QueryRow(ctx, query, id, workspaceID)

	var ep domain.Endpoint
	err := row.Scan(&ep.ID, &ep.WorkspaceID, &ep.Name, &ep.TargetURL, &ep.Interval, &ep.CreatedAt)
	if err != nil {
		// Explicit error wrapping. No silent failures.
		return nil, fmt.Errorf("failed to fetch endpoint by id and workspace: %w", err)
	}

	return &ep, nil
}

// GetActiveEndpoints fetches all endpoints that the concurrent worker pool needs to ping.
func (r *postgresEndpointRepo) GetActiveEndpoints(ctx context.Context) ([]domain.Endpoint, error) {
	query := `SELECT id, workspace_id, name, target_url, interval_seconds, created_at FROM endpoints`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []domain.Endpoint
	for rows.Next() {
		var ep domain.Endpoint
		if err := rows.Scan(&ep.ID, &ep.WorkspaceID, &ep.Name, &ep.TargetURL, &ep.Interval, &ep.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint row: %w", err)
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

// Save inserts a new endpoint using parameterized queries to eliminate SQL injection risks.
func (r *postgresEndpointRepo) Save(ctx context.Context, endpoint *domain.Endpoint) error {
	query := `
		INSERT INTO endpoints (id, workspace_id, name, target_url, interval_seconds, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query, 
		endpoint.ID, 
		endpoint.WorkspaceID, 
		endpoint.Name, 
		endpoint.TargetURL, 
		endpoint.Interval, 
		endpoint.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert endpoint: %w", err)
	}
	
	return nil
}