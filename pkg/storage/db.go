// Package storage provides TimescaleDB repositories for microcloud services.
package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new database connection pool
func New(ctx context.Context, cfg Config) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the connection pool
func (db *DB) Close() {
	db.pool.Close()
}

// Pool returns the underlying pgx pool for advanced usage
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Migrate runs database migrations
func (db *DB) Migrate(ctx context.Context) error {
	migrations := []string{
		// Enable TimescaleDB extension
		`CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE`,

		// Metrics hypertable
		`CREATE TABLE IF NOT EXISTS metrics (
			time TIMESTAMPTZ NOT NULL,
			tick_id BIGINT NOT NULL,
			node_id TEXT,
			service_id TEXT,
			metric_name TEXT NOT NULL,
			metric_value DOUBLE PRECISION NOT NULL,
			labels JSONB DEFAULT '{}'
		)`,
		`SELECT create_hypertable('metrics', 'time', if_not_exists => TRUE)`,

		// Incidents table
		`CREATE TABLE IF NOT EXISTS incidents (
			id UUID PRIMARY KEY,
			detected_at TIMESTAMPTZ NOT NULL,
			tick_id BIGINT NOT NULL,
			severity INT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			source_service TEXT,
			affected_ids TEXT[],
			rule_name TEXT,
			metrics JSONB,
			resolved BOOLEAN DEFAULT FALSE,
			resolved_at TIMESTAMPTZ
		)`,

		// Actions table
		`CREATE TABLE IF NOT EXISTS actions (
			id UUID PRIMARY KEY,
			incident_id UUID REFERENCES incidents(id),
			proposed_at_tick BIGINT NOT NULL,
			action_type INT NOT NULL,
			target_id TEXT NOT NULL,
			status INT NOT NULL,
			reason TEXT,
			parameters JSONB,
			created_at TIMESTAMPTZ NOT NULL,
			executed_at TIMESTAMPTZ,
			result_message TEXT
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_metrics_node ON metrics (node_id, time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_service ON metrics (service_id, time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents (severity, detected_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_actions_status ON actions (status, created_at DESC)`,
	}

	for _, migration := range migrations {
		if _, err := db.pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
