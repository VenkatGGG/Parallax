package bus

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"

	opsv1 "github.com/microcloud/gen/go/ops/v1"
	simv1 "github.com/microcloud/gen/go/sim/v1"
)

// MetricHandler handles incoming metric snapshots
type MetricHandler func(ctx context.Context, snapshot *simv1.MetricSnapshot) error

// SimEventHandler handles incoming simulation events
type SimEventHandler func(ctx context.Context, event *simv1.SimulationEvent) error

// IncidentHandler handles incoming incidents
type IncidentHandler func(ctx context.Context, incident *opsv1.Incident) error

// ActionHandler handles incoming actions
type ActionHandler func(ctx context.Context, action *opsv1.Action) error

// CommandHandler handles incoming action commands
type CommandHandler func(ctx context.Context, cmd *opsv1.ApplyActionCommand) error

// Subscriber provides typed subscription methods
type Subscriber struct {
	bus *Bus
}

// NewSubscriber creates a new typed subscriber
func NewSubscriber(bus *Bus) *Subscriber {
	return &Subscriber{bus: bus}
}

// SubscribeMetrics subscribes to sim.metrics with a durable consumer
func (s *Subscriber) SubscribeMetrics(ctx context.Context, consumerName string, handler MetricHandler) (jetstream.ConsumeContext, error) {
	return s.subscribe(ctx, SubjectSimMetrics, consumerName, func(ctx context.Context, data []byte) error {
		var msg simv1.MetricSnapshot
		if err := proto.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("unmarshal metric: %w", err)
		}
		return handler(ctx, &msg)
	})
}

// SubscribeSimEvents subscribes to sim.events with a durable consumer
func (s *Subscriber) SubscribeSimEvents(ctx context.Context, consumerName string, handler SimEventHandler) (jetstream.ConsumeContext, error) {
	return s.subscribe(ctx, SubjectSimEvents, consumerName, func(ctx context.Context, data []byte) error {
		var msg simv1.SimulationEvent
		if err := proto.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("unmarshal sim event: %w", err)
		}
		return handler(ctx, &msg)
	})
}

// SubscribeIncidents subscribes to ops.incidents with a durable consumer
func (s *Subscriber) SubscribeIncidents(ctx context.Context, consumerName string, handler IncidentHandler) (jetstream.ConsumeContext, error) {
	return s.subscribe(ctx, SubjectOpsIncidents, consumerName, func(ctx context.Context, data []byte) error {
		var msg opsv1.Incident
		if err := proto.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("unmarshal incident: %w", err)
		}
		return handler(ctx, &msg)
	})
}

// SubscribeActions subscribes to ops.actions with a durable consumer
func (s *Subscriber) SubscribeActions(ctx context.Context, consumerName string, handler ActionHandler) (jetstream.ConsumeContext, error) {
	return s.subscribe(ctx, SubjectOpsActions, consumerName, func(ctx context.Context, data []byte) error {
		var msg opsv1.Action
		if err := proto.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("unmarshal action: %w", err)
		}
		return handler(ctx, &msg)
	})
}

// SubscribeCommands subscribes to ops.commands with a durable consumer
func (s *Subscriber) SubscribeCommands(ctx context.Context, consumerName string, handler CommandHandler) (jetstream.ConsumeContext, error) {
	return s.subscribe(ctx, SubjectOpsCommands, consumerName, func(ctx context.Context, data []byte) error {
		var msg opsv1.ApplyActionCommand
		if err := proto.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("unmarshal command: %w", err)
		}
		return handler(ctx, &msg)
	})
}

func (s *Subscriber) subscribe(ctx context.Context, subject, consumerName string, handler func(context.Context, []byte) error) (jetstream.ConsumeContext, error) {
	consumer, err := s.bus.js.CreateOrUpdateConsumer(ctx, s.bus.cfg.StreamName, jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("create consumer %s: %w", consumerName, err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(ctx, msg.Data()); err != nil {
			msg.Nak()
			return
		}
		msg.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("consume %s: %w", subject, err)
	}

	return cc, nil
}
