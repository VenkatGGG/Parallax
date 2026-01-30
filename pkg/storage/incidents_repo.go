package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// IncidentRow represents an incident in the database
type IncidentRow struct {
	ID            string
	DetectedAt    time.Time
	TickID        int64
	Severity      int
	Title         string
	Description   string
	SourceService string
	AffectedIDs   []string
	RuleName      string
	Metrics       map[string]float64
	Resolved      bool
	ResolvedAt    *time.Time
}

// IncidentsRepository handles incident persistence
type IncidentsRepository struct {
	db *DB
}

// NewIncidentsRepository creates a new incidents repository
func NewIncidentsRepository(db *DB) *IncidentsRepository {
	return &IncidentsRepository{db: db}
}

// Create inserts a new incident
func (r *IncidentsRepository) Create(ctx context.Context, incident IncidentRow) error {
	query := `
		INSERT INTO incidents (id, detected_at, tick_id, severity, title, description,
							   source_service, affected_ids, rule_name, metrics, resolved, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.pool.Exec(ctx, query,
		incident.ID, incident.DetectedAt, incident.TickID, incident.Severity,
		incident.Title, incident.Description, incident.SourceService,
		incident.AffectedIDs, incident.RuleName, incident.Metrics,
		incident.Resolved, incident.ResolvedAt,
	)
	if err != nil {
		return fmt.Errorf("create incident: %w", err)
	}
	return nil
}

// GetByID retrieves an incident by ID
func (r *IncidentsRepository) GetByID(ctx context.Context, id string) (*IncidentRow, error) {
	query := `
		SELECT id, detected_at, tick_id, severity, title, description,
			   source_service, affected_ids, rule_name, metrics, resolved, resolved_at
		FROM incidents WHERE id = $1
	`
	var i IncidentRow
	err := r.db.pool.QueryRow(ctx, query, id).Scan(
		&i.ID, &i.DetectedAt, &i.TickID, &i.Severity, &i.Title, &i.Description,
		&i.SourceService, &i.AffectedIDs, &i.RuleName, &i.Metrics, &i.Resolved, &i.ResolvedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}
	return &i, nil
}

// ListUnresolved returns all unresolved incidents
func (r *IncidentsRepository) ListUnresolved(ctx context.Context, limit int) ([]IncidentRow, error) {
	query := `
		SELECT id, detected_at, tick_id, severity, title, description,
			   source_service, affected_ids, rule_name, metrics, resolved, resolved_at
		FROM incidents
		WHERE resolved = FALSE
		ORDER BY severity DESC, detected_at DESC
		LIMIT $1
	`
	return r.queryIncidents(ctx, query, limit)
}

// ListRecent returns recent incidents
func (r *IncidentsRepository) ListRecent(ctx context.Context, limit int) ([]IncidentRow, error) {
	query := `
		SELECT id, detected_at, tick_id, severity, title, description,
			   source_service, affected_ids, rule_name, metrics, resolved, resolved_at
		FROM incidents
		ORDER BY detected_at DESC
		LIMIT $1
	`
	return r.queryIncidents(ctx, query, limit)
}

// ListBySeverity returns incidents filtered by minimum severity
func (r *IncidentsRepository) ListBySeverity(ctx context.Context, minSeverity int, limit int) ([]IncidentRow, error) {
	query := `
		SELECT id, detected_at, tick_id, severity, title, description,
			   source_service, affected_ids, rule_name, metrics, resolved, resolved_at
		FROM incidents
		WHERE severity >= $1
		ORDER BY severity DESC, detected_at DESC
		LIMIT $2
	`
	return r.queryIncidents(ctx, query, minSeverity, limit)
}

// MarkResolved marks an incident as resolved
func (r *IncidentsRepository) MarkResolved(ctx context.Context, id string, resolvedAt time.Time) error {
	query := `UPDATE incidents SET resolved = TRUE, resolved_at = $2 WHERE id = $1`
	_, err := r.db.pool.Exec(ctx, query, id, resolvedAt)
	if err != nil {
		return fmt.Errorf("mark resolved: %w", err)
	}
	return nil
}

// CountUnresolved returns the count of unresolved incidents
func (r *IncidentsRepository) CountUnresolved(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM incidents WHERE resolved = FALSE`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unresolved: %w", err)
	}
	return count, nil
}

func (r *IncidentsRepository) queryIncidents(ctx context.Context, query string, args ...any) ([]IncidentRow, error) {
	rows, err := r.db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var results []IncidentRow
	for rows.Next() {
		var i IncidentRow
		if err := rows.Scan(
			&i.ID, &i.DetectedAt, &i.TickID, &i.Severity, &i.Title, &i.Description,
			&i.SourceService, &i.AffectedIDs, &i.RuleName, &i.Metrics, &i.Resolved, &i.ResolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		results = append(results, i)
	}
	return results, rows.Err()
}
