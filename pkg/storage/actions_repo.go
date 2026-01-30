package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ActionRow represents an action in the database
type ActionRow struct {
	ID             string
	IncidentID     string
	ProposedAtTick int64
	ActionType     int
	TargetID       string
	Status         int
	Reason         string
	Parameters     map[string]string
	CreatedAt      time.Time
	ExecutedAt     *time.Time
	ResultMessage  string
}

// ActionsRepository handles action persistence
type ActionsRepository struct {
	db *DB
}

// NewActionsRepository creates a new actions repository
func NewActionsRepository(db *DB) *ActionsRepository {
	return &ActionsRepository{db: db}
}

// Create inserts a new action
func (r *ActionsRepository) Create(ctx context.Context, action ActionRow) error {
	query := `
		INSERT INTO actions (id, incident_id, proposed_at_tick, action_type, target_id,
							status, reason, parameters, created_at, executed_at, result_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.pool.Exec(ctx, query,
		action.ID, action.IncidentID, action.ProposedAtTick, action.ActionType,
		action.TargetID, action.Status, action.Reason, action.Parameters,
		action.CreatedAt, action.ExecutedAt, action.ResultMessage,
	)
	if err != nil {
		return fmt.Errorf("create action: %w", err)
	}
	return nil
}

// GetByID retrieves an action by ID
func (r *ActionsRepository) GetByID(ctx context.Context, id string) (*ActionRow, error) {
	query := `
		SELECT id, incident_id, proposed_at_tick, action_type, target_id,
			   status, reason, parameters, created_at, executed_at, result_message
		FROM actions WHERE id = $1
	`
	var a ActionRow
	err := r.db.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.IncidentID, &a.ProposedAtTick, &a.ActionType, &a.TargetID,
		&a.Status, &a.Reason, &a.Parameters, &a.CreatedAt, &a.ExecutedAt, &a.ResultMessage,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get action: %w", err)
	}
	return &a, nil
}

// ListPending returns all pending actions (status = 1)
func (r *ActionsRepository) ListPending(ctx context.Context, limit int) ([]ActionRow, error) {
	query := `
		SELECT id, incident_id, proposed_at_tick, action_type, target_id,
			   status, reason, parameters, created_at, executed_at, result_message
		FROM actions
		WHERE status = 1
		ORDER BY created_at ASC
		LIMIT $1
	`
	return r.queryActions(ctx, query, limit)
}

// ListByStatus returns actions filtered by status
func (r *ActionsRepository) ListByStatus(ctx context.Context, status int, limit int) ([]ActionRow, error) {
	query := `
		SELECT id, incident_id, proposed_at_tick, action_type, target_id,
			   status, reason, parameters, created_at, executed_at, result_message
		FROM actions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	return r.queryActions(ctx, query, status, limit)
}

// ListByIncident returns all actions for an incident
func (r *ActionsRepository) ListByIncident(ctx context.Context, incidentID string) ([]ActionRow, error) {
	query := `
		SELECT id, incident_id, proposed_at_tick, action_type, target_id,
			   status, reason, parameters, created_at, executed_at, result_message
		FROM actions
		WHERE incident_id = $1
		ORDER BY created_at ASC
	`
	return r.queryActions(ctx, query, incidentID)
}

// ListRecent returns recent actions
func (r *ActionsRepository) ListRecent(ctx context.Context, limit int) ([]ActionRow, error) {
	query := `
		SELECT id, incident_id, proposed_at_tick, action_type, target_id,
			   status, reason, parameters, created_at, executed_at, result_message
		FROM actions
		ORDER BY created_at DESC
		LIMIT $1
	`
	return r.queryActions(ctx, query, limit)
}

// UpdateStatus updates an action's status
func (r *ActionsRepository) UpdateStatus(ctx context.Context, id string, status int, resultMessage string) error {
	query := `UPDATE actions SET status = $2, result_message = $3, executed_at = $4 WHERE id = $1`
	_, err := r.db.pool.Exec(ctx, query, id, status, resultMessage, time.Now())
	if err != nil {
		return fmt.Errorf("update action status: %w", err)
	}
	return nil
}

// Approve marks an action as approved (status = 2)
func (r *ActionsRepository) Approve(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, 2, "")
}

// Reject marks an action as rejected (status = 3)
func (r *ActionsRepository) Reject(ctx context.Context, id string, reason string) error {
	return r.UpdateStatus(ctx, id, 3, reason)
}

// MarkExecuting marks an action as executing (status = 4)
func (r *ActionsRepository) MarkExecuting(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, 4, "")
}

// MarkCompleted marks an action as completed (status = 5)
func (r *ActionsRepository) MarkCompleted(ctx context.Context, id string, resultMessage string) error {
	return r.UpdateStatus(ctx, id, 5, resultMessage)
}

// MarkFailed marks an action as failed (status = 6)
func (r *ActionsRepository) MarkFailed(ctx context.Context, id string, errorMessage string) error {
	return r.UpdateStatus(ctx, id, 6, errorMessage)
}

func (r *ActionsRepository) queryActions(ctx context.Context, query string, args ...any) ([]ActionRow, error) {
	rows, err := r.db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query actions: %w", err)
	}
	defer rows.Close()

	var results []ActionRow
	for rows.Next() {
		var a ActionRow
		if err := rows.Scan(
			&a.ID, &a.IncidentID, &a.ProposedAtTick, &a.ActionType, &a.TargetID,
			&a.Status, &a.Reason, &a.Parameters, &a.CreatedAt, &a.ExecutedAt, &a.ResultMessage,
		); err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		results = append(results, a)
	}
	return results, rows.Err()
}
