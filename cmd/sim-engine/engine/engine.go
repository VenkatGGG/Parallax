package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/microcloud/bus"
	commonv1 "github.com/microcloud/gen/go/common/v1"
	simv1 "github.com/microcloud/gen/go/sim/v1"
)

const (
	DefaultTickInterval = 100 * time.Millisecond
)

// Engine runs the simulation loop
type Engine struct {
	state     *State
	publisher *bus.Publisher
	log       *slog.Logger

	tickInterval time.Duration
}

// New creates a new simulation engine
func New(publisher *bus.Publisher, log *slog.Logger) *Engine {
	return &Engine{
		state:        NewState(),
		publisher:    publisher,
		log:          log,
		tickInterval: DefaultTickInterval,
	}
}

// State returns the simulation state for the control server
func (e *Engine) State() *State {
	return e.state
}

// Run starts the simulation loop (blocking)
func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(e.tickInterval)
	defer ticker.Stop()

	e.log.Info("simulation engine started", "tick_interval", e.tickInterval)

	for {
		select {
		case <-ctx.Done():
			e.log.Info("simulation engine stopped")
			return ctx.Err()
		case <-ticker.C:
			if e.state.GetSimState() != commonv1.SimulationState_SIMULATION_STATE_RUNNING {
				continue
			}

			e.state.Tick(e.tickInterval)
			snapshot := e.state.Snapshot()

			if err := e.publisher.PublishMetricSnapshot(ctx, snapshot); err != nil {
				e.log.Error("failed to publish metrics", "error", err)
			}

			if e.state.GetTickID()%100 == 0 {
				e.log.Debug("tick", "tick_id", snapshot.Timestamp.TickId, "nodes", len(snapshot.Nodes), "services", len(snapshot.Services))
			}
		}
	}
}

// ApplyCommand applies an action command to the simulation
func (e *Engine) ApplyCommand(ctx context.Context, actionType commonv1.ActionType, targetID string, params map[string]string) (*simv1.SimulationEvent, error) {
	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	event := &simv1.SimulationEvent{
		Timestamp: &commonv1.SimulationTimestamp{
			TickId:        e.state.tickID,
			WallTimeUnixMs: time.Now().UnixMilli(),
			SimTimeUnixMs:  e.state.simTimeUnixMs,
		},
		TargetId: targetID,
		Metadata: params,
	}

	switch actionType {
	case commonv1.ActionType_ACTION_TYPE_RESTART_SERVICE:
		if svc, ok := e.state.services[targetID]; ok {
			svc.Health = commonv1.ServiceHealth_SERVICE_HEALTH_HEALTHY
			svc.ErrorRatePercent = 0.1
			svc.LatencyP50Ms = 5
			svc.LatencyP99Ms = 20
			event.EventType = "service_restarted"
			event.Description = "Service restarted successfully"
		}

	case commonv1.ActionType_ACTION_TYPE_SCALE_UP:
		if svc, ok := e.state.services[targetID]; ok {
			svc.ReplicaCount++
			svc.DesiredReplicas = svc.ReplicaCount
			event.EventType = "service_scaled_up"
			event.Description = "Service scaled up"
		}

	case commonv1.ActionType_ACTION_TYPE_SCALE_DOWN:
		if svc, ok := e.state.services[targetID]; ok && svc.ReplicaCount > 1 {
			svc.ReplicaCount--
			svc.DesiredReplicas = svc.ReplicaCount
			event.EventType = "service_scaled_down"
			event.Description = "Service scaled down"
		}

	case commonv1.ActionType_ACTION_TYPE_DRAIN_NODE:
		if node, ok := e.state.nodes[targetID]; ok {
			node.Status = commonv1.NodeStatus_NODE_STATUS_OFFLINE
			node.RunningServices = 0
			event.EventType = "node_drained"
			event.Description = "Node drained and offline"
		}

	case commonv1.ActionType_ACTION_TYPE_REBALANCE_TRAFFIC:
		for _, svc := range e.state.services {
			svc.RequestsPerSecond = svc.RequestsPerSecond * 0.9
		}
		event.EventType = "traffic_rebalanced"
		event.Description = "Traffic rebalanced across services"
	}

	if err := e.publisher.PublishSimulationEvent(ctx, event); err != nil {
		e.log.Error("failed to publish event", "error", err)
	}

	return event, nil
}
