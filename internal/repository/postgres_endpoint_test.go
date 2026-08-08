package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"latencyops/internal/domain"
	"latencyops/internal/repository"
)

func TestPostgresEndpointRepo_GetByID_TenantIsolation(t *testing.T) {
	// Initialize pgxmock to intercept database calls
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := repository.NewPostgresEndpointRepo(mock)

	endpointID := "test-endpoint-id" // UUIDv4 or KSUID
	workspaceID := "test-workspace-id"
	now := time.Now()

	// ASSERTION: Verify the query includes the strict OWASP BOLA mitigation parameters
	// The regex checks that both id and workspace_id are being filtered.
	mock.ExpectQuery(`SELECT id, workspace_id, name, target_url, interval_seconds, created_at FROM endpoints WHERE id = \$1 AND workspace_id = \$2`).
		WithArgs(endpointID, workspaceID).
		WillReturnRows(mock.NewRows([]string{"id", "workspace_id", "name", "target_url", "interval_seconds", "created_at"}).
			AddRow(endpointID, workspaceID, "Production API", "https://api.example.com", 60, now))

	ctx := context.Background()
	ep, err := repo.GetByID(ctx, endpointID, workspaceID)

	if err != nil {
		t.Errorf("error was not expected while getting endpoint: %s", err)
	}

	if ep.ID != endpointID {
		t.Errorf("expected endpoint ID %s, got %s", endpointID, ep.ID)
	}

	if ep.WorkspaceID != workspaceID {
		t.Errorf("expected workspace ID %s, got %s - Tenant isolation failed!", workspaceID, ep.WorkspaceID)
	}

	// Ensure all expectations dictated by our strict testing criteria were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestPostgresEndpointRepo_Save_ParameterizedQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := repository.NewPostgresEndpointRepo(mock)
	now := time.Now()

	endpoint := &domain.Endpoint{
		ID:          "new-uuid",
		WorkspaceID: "workspace-uuid",
		Name:        "Staging DB",
		TargetURL:   "https://staging.example.com",
		Interval:    30,
		CreatedAt:   now,
	}

	// ASSERTION: Ensure INSERT uses all 6 parameterized arguments to prevent SQL injection
	mock.ExpectExec(`INSERT INTO endpoints`).
		WithArgs(endpoint.ID, endpoint.WorkspaceID, endpoint.Name, endpoint.TargetURL, endpoint.Interval, endpoint.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := context.Background()
	err = repo.Save(ctx, endpoint)

	if err != nil {
		t.Errorf("error was not expected while saving endpoint: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}