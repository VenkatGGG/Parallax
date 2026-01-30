package bus

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	opsv1 "github.com/microcloud/gen/go/ops/v1"
	simv1 "github.com/microcloud/gen/go/sim/v1"
)

// Publisher provides typed publishing methods
type Publisher struct {
	bus *Bus
}

// NewPublisher creates a new typed publisher
func NewPublisher(bus *Bus) *Publisher {
	return &Publisher{bus: bus}
}

// PublishMetricSnapshot publishes a metric snapshot to sim.metrics
func (p *Publisher) PublishMetricSnapshot(ctx context.Context, snapshot *simv1.MetricSnapshot) error {
	return p.publish(ctx, SubjectSimMetrics, snapshot)
}

// PublishSimulationEvent publishes a simulation event to sim.events
func (p *Publisher) PublishSimulationEvent(ctx context.Context, event *simv1.SimulationEvent) error {
	return p.publish(ctx, SubjectSimEvents, event)
}

// PublishIncident publishes an incident to ops.incidents
func (p *Publisher) PublishIncident(ctx context.Context, incident *opsv1.Incident) error {
	return p.publish(ctx, SubjectOpsIncidents, incident)
}

// PublishAction publishes a proposed action to ops.actions
func (p *Publisher) PublishAction(ctx context.Context, action *opsv1.Action) error {
	return p.publish(ctx, SubjectOpsActions, action)
}

// PublishCommand publishes an action command to ops.commands
func (p *Publisher) PublishCommand(ctx context.Context, cmd *opsv1.ApplyActionCommand) error {
	return p.publish(ctx, SubjectOpsCommands, cmd)
}

func (p *Publisher) publish(ctx context.Context, subject string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal proto: %w", err)
	}

	_, err = p.bus.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
