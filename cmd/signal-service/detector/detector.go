package detector

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/microcloud/bus"
	commonv1 "github.com/microcloud/gen/go/common/v1"
	opsv1 "github.com/microcloud/gen/go/ops/v1"
	simv1 "github.com/microcloud/gen/go/sim/v1"
	"github.com/microcloud/storage"
)

// Detector monitors metrics and detects incidents
type Detector struct {
	publisher   *bus.Publisher
	metricsRepo *storage.MetricsRepository
	log         *slog.Logger
	rules       []Rule

	mu             sync.Mutex
	windows        map[string]*metricWindow
	activeIncidents map[string]bool
}

type metricWindow struct {
	values    []float64
	timestamps []time.Time
}

// New creates a new detector
func New(publisher *bus.Publisher, metricsRepo *storage.MetricsRepository, log *slog.Logger) *Detector {
	return &Detector{
		publisher:       publisher,
		metricsRepo:     metricsRepo,
		log:             log,
		rules:           DefaultRules(),
		windows:         make(map[string]*metricWindow),
		activeIncidents: make(map[string]bool),
	}
}

// ProcessSnapshot processes a metric snapshot
func (d *Detector) ProcessSnapshot(ctx context.Context, snapshot *simv1.MetricSnapshot) error {
	now := time.Now()
	tickID := snapshot.Timestamp.TickId

	var metricsToStore []storage.MetricRow

	for _, node := range snapshot.Nodes {
		nodeID := node.Id.Value

		metricsToStore = append(metricsToStore,
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				NodeID:      &nodeID,
				MetricName:  "cpu_usage_percent",
				MetricValue: node.CpuUsagePercent,
			},
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				NodeID:      &nodeID,
				MetricName:  "memory_usage_percent",
				MetricValue: node.MemoryUsagePercent,
			},
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				NodeID:      &nodeID,
				MetricName:  "disk_usage_percent",
				MetricValue: node.DiskUsagePercent,
			},
		)

		d.checkRulesForEntity(ctx, "node", nodeID, map[string]float64{
			"cpu_usage_percent":    node.CpuUsagePercent,
			"memory_usage_percent": node.MemoryUsagePercent,
			"disk_usage_percent":   node.DiskUsagePercent,
		}, tickID)
	}

	for _, svc := range snapshot.Services {
		svcID := svc.Id.Value

		metricsToStore = append(metricsToStore,
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				ServiceID:   &svcID,
				MetricName:  "requests_per_second",
				MetricValue: svc.RequestsPerSecond,
			},
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				ServiceID:   &svcID,
				MetricName:  "error_rate_percent",
				MetricValue: svc.ErrorRatePercent,
			},
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				ServiceID:   &svcID,
				MetricName:  "latency_p50_ms",
				MetricValue: svc.LatencyP50Ms,
			},
			storage.MetricRow{
				Time:        now,
				TickID:      tickID,
				ServiceID:   &svcID,
				MetricName:  "latency_p99_ms",
				MetricValue: svc.LatencyP99Ms,
			},
		)

		d.checkRulesForEntity(ctx, "service", svcID, map[string]float64{
			"error_rate_percent": svc.ErrorRatePercent,
			"latency_p50_ms":     svc.LatencyP50Ms,
			"latency_p99_ms":     svc.LatencyP99Ms,
		}, tickID)
	}

	if err := d.metricsRepo.BatchInsert(ctx, metricsToStore); err != nil {
		d.log.Error("failed to store metrics", "error", err)
	}

	return nil
}

func (d *Detector) checkRulesForEntity(ctx context.Context, entityType, entityID string, metrics map[string]float64, tickID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	for _, rule := range d.rules {
		value, ok := metrics[rule.MetricName]
		if !ok {
			continue
		}

		windowKey := fmt.Sprintf("%s:%s:%s", entityType, entityID, rule.Name)
		window, exists := d.windows[windowKey]
		if !exists {
			window = &metricWindow{
				values:     make([]float64, 0, 100),
				timestamps: make([]time.Time, 0, 100),
			}
			d.windows[windowKey] = window
		}

		window.values = append(window.values, value)
		window.timestamps = append(window.timestamps, now)

		cutoff := now.Add(-time.Duration(rule.WindowSeconds) * time.Second)
		startIdx := 0
		for i, ts := range window.timestamps {
			if ts.After(cutoff) {
				startIdx = i
				break
			}
		}
		window.values = window.values[startIdx:]
		window.timestamps = window.timestamps[startIdx:]

		if len(window.values) < 3 {
			continue
		}

		breachCount := 0
		for _, v := range window.values {
			if rule.Evaluate(v) {
				breachCount++
			}
		}

		breachRatio := float64(breachCount) / float64(len(window.values))
		incidentKey := fmt.Sprintf("%s:%s:%s", entityType, entityID, rule.Name)

		if breachRatio > 0.7 && !d.activeIncidents[incidentKey] {
			d.activeIncidents[incidentKey] = true

			incident := &opsv1.Incident{
				Id:            &commonv1.UUID{Value: randomUUID()},
				DetectedAt:    &commonv1.SimulationTimestamp{TickId: tickID, WallTimeUnixMs: now.UnixMilli()},
				Severity:      rule.Severity,
				Title:         fmt.Sprintf("%s: %s on %s %s", rule.Name, rule.MetricName, entityType, entityID[:8]),
				Description:   fmt.Sprintf("%s breached threshold %.2f (current: %.2f) for %d seconds", rule.MetricName, rule.Threshold, value, rule.WindowSeconds),
				SourceService: "signal-service",
				AffectedIds:   []string{entityID},
				RuleName:      rule.Name,
				Metrics:       map[string]float64{rule.MetricName: value},
				Resolved:      false,
			}

			if err := d.publisher.PublishIncident(ctx, incident); err != nil {
				d.log.Error("failed to publish incident", "error", err)
			} else {
				d.log.Warn("incident detected", "rule", rule.Name, "entity", entityID[:8], "severity", rule.Severity)
			}
		} else if breachRatio < 0.3 && d.activeIncidents[incidentKey] {
			delete(d.activeIncidents, incidentKey)
			d.log.Info("incident resolved", "rule", rule.Name, "entity", entityID[:8])
		}
	}
}

func randomUUID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 4))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
