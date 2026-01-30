package decider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/microcloud/bus"
	commonv1 "github.com/microcloud/gen/go/common/v1"
	opsv1 "github.com/microcloud/gen/go/ops/v1"
	"github.com/microcloud/storage"
)

// Decider processes incidents and proposes actions
type Decider struct {
	publisher    *bus.Publisher
	actionsRepo  *storage.ActionsRepository
	incidentsRepo *storage.IncidentsRepository
	log          *slog.Logger

	mu               sync.Mutex
	recentActions    map[string]time.Time
	cooldownDuration time.Duration
}

// New creates a new decider
func New(publisher *bus.Publisher, actionsRepo *storage.ActionsRepository, incidentsRepo *storage.IncidentsRepository, log *slog.Logger) *Decider {
	return &Decider{
		publisher:        publisher,
		actionsRepo:      actionsRepo,
		incidentsRepo:    incidentsRepo,
		log:              log,
		recentActions:    make(map[string]time.Time),
		cooldownDuration: 30 * time.Second,
	}
}

// ProcessIncident processes an incident and proposes actions
func (d *Decider) ProcessIncident(ctx context.Context, incident *opsv1.Incident) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.storeIncident(ctx, incident); err != nil {
		d.log.Error("failed to store incident", "error", err)
	}

	actionKey := fmt.Sprintf("%s:%s", incident.RuleName, incident.AffectedIds[0])
	if lastAction, ok := d.recentActions[actionKey]; ok {
		if time.Since(lastAction) < d.cooldownDuration {
			d.log.Debug("action cooldown active", "key", actionKey)
			return nil
		}
	}

	action := d.decideAction(incident)
	if action == nil {
		return nil
	}

	if err := d.storeAction(ctx, action); err != nil {
		d.log.Error("failed to store action", "error", err)
	}

	if err := d.publisher.PublishAction(ctx, action); err != nil {
		return fmt.Errorf("publish action: %w", err)
	}

	d.recentActions[actionKey] = time.Now()
	d.log.Info("action proposed",
		"action_type", action.ActionType,
		"target", action.TargetId,
		"reason", action.Reason,
	)

	return nil
}

func (d *Decider) decideAction(incident *opsv1.Incident) *opsv1.Action {
	if len(incident.AffectedIds) == 0 {
		return nil
	}

	targetID := incident.AffectedIds[0]
	tickID := incident.DetectedAt.TickId
	now := time.Now()

	action := &opsv1.Action{
		Id:             &commonv1.UUID{Value: randomUUID()},
		IncidentId:     incident.Id,
		ProposedAtTick: tickID,
		TargetId:       targetID,
		Status:         commonv1.ActionStatus_ACTION_STATUS_PENDING,
		Parameters:     make(map[string]string),
		CreatedAt: &commonv1.SimulationTimestamp{
			TickId:        tickID,
			WallTimeUnixMs: now.UnixMilli(),
		},
	}

	switch incident.RuleName {
	case "high_error_rate", "critical_error_rate":
		action.ActionType = commonv1.ActionType_ACTION_TYPE_RESTART_SERVICE
		action.Reason = fmt.Sprintf("Auto-restart due to %s (error rate: %.2f%%)",
			incident.RuleName, incident.Metrics["error_rate_percent"])

	case "high_cpu_usage", "critical_cpu_usage":
		if incident.Severity == commonv1.IncidentSeverity_INCIDENT_SEVERITY_CRITICAL {
			action.ActionType = commonv1.ActionType_ACTION_TYPE_SCALE_UP
			action.Reason = fmt.Sprintf("Scale up due to critical CPU (%.2f%%)",
				incident.Metrics["cpu_usage_percent"])
		} else {
			action.ActionType = commonv1.ActionType_ACTION_TYPE_REBALANCE_TRAFFIC
			action.Reason = fmt.Sprintf("Rebalance traffic due to high CPU (%.2f%%)",
				incident.Metrics["cpu_usage_percent"])
		}

	case "high_memory_usage":
		action.ActionType = commonv1.ActionType_ACTION_TYPE_RESTART_SERVICE
		action.Reason = fmt.Sprintf("Restart due to high memory usage (%.2f%%)",
			incident.Metrics["memory_usage_percent"])

	case "high_latency":
		action.ActionType = commonv1.ActionType_ACTION_TYPE_SCALE_UP
		action.Reason = fmt.Sprintf("Scale up due to high latency (%.2fms)",
			incident.Metrics["latency_p99_ms"])

	default:
		d.log.Debug("no action rule for incident", "rule", incident.RuleName)
		return nil
	}

	return action
}

func (d *Decider) storeIncident(ctx context.Context, incident *opsv1.Incident) error {
	row := storage.IncidentRow{
		ID:            incident.Id.Value,
		DetectedAt:    time.UnixMilli(incident.DetectedAt.WallTimeUnixMs),
		TickID:        incident.DetectedAt.TickId,
		Severity:      int(incident.Severity),
		Title:         incident.Title,
		Description:   incident.Description,
		SourceService: incident.SourceService,
		AffectedIDs:   incident.AffectedIds,
		RuleName:      incident.RuleName,
		Metrics:       incident.Metrics,
		Resolved:      incident.Resolved,
	}
	return d.incidentsRepo.Create(ctx, row)
}

func (d *Decider) storeAction(ctx context.Context, action *opsv1.Action) error {
	row := storage.ActionRow{
		ID:             action.Id.Value,
		IncidentID:     action.IncidentId.Value,
		ProposedAtTick: action.ProposedAtTick,
		ActionType:     int(action.ActionType),
		TargetID:       action.TargetId,
		Status:         int(action.Status),
		Reason:         action.Reason,
		Parameters:     action.Parameters,
		CreatedAt:      time.UnixMilli(action.CreatedAt.WallTimeUnixMs),
	}
	return d.actionsRepo.Create(ctx, row)
}

func randomUUID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 4))
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
