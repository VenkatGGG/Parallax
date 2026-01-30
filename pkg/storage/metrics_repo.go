package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// MetricRow represents a single metric data point
type MetricRow struct {
	Time        time.Time
	TickID      int64
	NodeID      *string
	ServiceID   *string
	MetricName  string
	MetricValue float64
	Labels      map[string]string
}

// MetricsRepository handles metric persistence
type MetricsRepository struct {
	db *DB
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *DB) *MetricsRepository {
	return &MetricsRepository{db: db}
}

// BatchInsert efficiently inserts multiple metrics
func (r *MetricsRepository) BatchInsert(ctx context.Context, metrics []MetricRow) error {
	if len(metrics) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, m := range metrics {
		batch.Queue(
			`INSERT INTO metrics (time, tick_id, node_id, service_id, metric_name, metric_value, labels)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			m.Time, m.TickID, m.NodeID, m.ServiceID, m.MetricName, m.MetricValue, m.Labels,
		)
	}

	br := r.db.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(metrics); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch insert metric %d: %w", i, err)
		}
	}
	return nil
}

// QueryByTimeRange retrieves metrics within a time range
func (r *MetricsRepository) QueryByTimeRange(ctx context.Context, start, end time.Time, metricName string, limit int) ([]MetricRow, error) {
	query := `
		SELECT time, tick_id, node_id, service_id, metric_name, metric_value, labels
		FROM metrics
		WHERE time >= $1 AND time < $2 AND metric_name = $3
		ORDER BY time DESC
		LIMIT $4
	`

	rows, err := r.db.pool.Query(ctx, query, start, end, metricName, limit)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var results []MetricRow
	for rows.Next() {
		var m MetricRow
		if err := rows.Scan(&m.Time, &m.TickID, &m.NodeID, &m.ServiceID, &m.MetricName, &m.MetricValue, &m.Labels); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetLatestByNode retrieves the latest metrics for a specific node
func (r *MetricsRepository) GetLatestByNode(ctx context.Context, nodeID string, limit int) ([]MetricRow, error) {
	query := `
		SELECT time, tick_id, node_id, service_id, metric_name, metric_value, labels
		FROM metrics
		WHERE node_id = $1
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := r.db.pool.Query(ctx, query, nodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("query node metrics: %w", err)
	}
	defer rows.Close()

	var results []MetricRow
	for rows.Next() {
		var m MetricRow
		if err := rows.Scan(&m.Time, &m.TickID, &m.NodeID, &m.ServiceID, &m.MetricName, &m.MetricValue, &m.Labels); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetLatestByService retrieves the latest metrics for a specific service
func (r *MetricsRepository) GetLatestByService(ctx context.Context, serviceID string, limit int) ([]MetricRow, error) {
	query := `
		SELECT time, tick_id, node_id, service_id, metric_name, metric_value, labels
		FROM metrics
		WHERE service_id = $1
		ORDER BY time DESC
		LIMIT $2
	`

	rows, err := r.db.pool.Query(ctx, query, serviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("query service metrics: %w", err)
	}
	defer rows.Close()

	var results []MetricRow
	for rows.Next() {
		var m MetricRow
		if err := rows.Scan(&m.Time, &m.TickID, &m.NodeID, &m.ServiceID, &m.MetricName, &m.MetricValue, &m.Labels); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// AggregatedMetric represents a time-bucketed aggregation
type AggregatedMetric struct {
	Bucket      time.Time
	AvgValue    float64
	MinValue    float64
	MaxValue    float64
	SampleCount int64
}

// Aggregate returns aggregated metrics using TimescaleDB time_bucket
func (r *MetricsRepository) Aggregate(ctx context.Context, metricName string, interval string, start, end time.Time) ([]AggregatedMetric, error) {
	query := `
		SELECT time_bucket($1::interval, time) AS bucket,
			   AVG(metric_value) AS avg_value,
			   MIN(metric_value) AS min_value,
			   MAX(metric_value) AS max_value,
			   COUNT(*) AS sample_count
		FROM metrics
		WHERE metric_name = $2 AND time >= $3 AND time < $4
		GROUP BY bucket
		ORDER BY bucket DESC
	`

	rows, err := r.db.pool.Query(ctx, query, interval, metricName, start, end)
	if err != nil {
		return nil, fmt.Errorf("aggregate metrics: %w", err)
	}
	defer rows.Close()

	var results []AggregatedMetric
	for rows.Next() {
		var m AggregatedMetric
		if err := rows.Scan(&m.Bucket, &m.AvgValue, &m.MinValue, &m.MaxValue, &m.SampleCount); err != nil {
			return nil, fmt.Errorf("scan aggregate: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}
