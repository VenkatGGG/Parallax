// Package bus provides a typed NATS JetStream wrapper for microcloud services.
package bus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Subjects for the event bus
const (
	SubjectSimMetrics   = "sim.metrics"
	SubjectSimEvents    = "sim.events"
	SubjectOpsIncidents = "ops.incidents"
	SubjectOpsActions   = "ops.actions"
	SubjectOpsCommands  = "ops.commands"
)

// Config holds NATS connection configuration
type Config struct {
	URL             string
	MaxReconnects   int
	ReconnectWait   time.Duration
	StreamName      string
	RetentionPolicy string
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		URL:             "nats://localhost:4222",
		MaxReconnects:   -1,
		ReconnectWait:   2 * time.Second,
		StreamName:      "MICROCLOUD",
		RetentionPolicy: "limits",
	}
}

// Bus wraps NATS JetStream with typed publishing and subscribing
type Bus struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	cfg    Config
	mu     sync.RWMutex
	closed bool

	onDisconnect func(error)
	onReconnect  func()
}

// Option configures the Bus
type Option func(*Bus)

// WithDisconnectHandler sets a callback for disconnect events
func WithDisconnectHandler(fn func(error)) Option {
	return func(b *Bus) {
		b.onDisconnect = fn
	}
}

// WithReconnectHandler sets a callback for reconnect events
func WithReconnectHandler(fn func()) Option {
	return func(b *Bus) {
		b.onReconnect = fn
	}
}

// New creates a new Bus with automatic reconnection handling
func New(ctx context.Context, cfg Config, opts ...Option) (*Bus, error) {
	b := &Bus{cfg: cfg}
	for _, opt := range opts {
		opt(b)
	}

	natsOpts := []nats.Option{
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if b.onDisconnect != nil && err != nil {
				b.onDisconnect(err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			if b.onReconnect != nil {
				b.onReconnect()
			}
		}),
	}

	nc, err := nats.Connect(cfg.URL, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	streamCfg := jetstream.StreamConfig{
		Name:      cfg.StreamName,
		Subjects:  []string{"sim.>", "ops.>"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.FileStorage,
	}

	stream, err := js.CreateOrUpdateStream(ctx, streamCfg)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create stream: %w", err)
	}

	b.nc = nc
	b.js = js
	b.stream = stream

	return b, nil
}

// Close gracefully shuts down the bus
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	b.nc.Close()
	return nil
}

// IsConnected returns true if connected to NATS
func (b *Bus) IsConnected() bool {
	return b.nc.IsConnected()
}

// JetStream returns the underlying JetStream context for advanced usage
func (b *Bus) JetStream() jetstream.JetStream {
	return b.js
}

// StreamName returns the configured stream name
func (b *Bus) StreamName() string {
	return b.cfg.StreamName
}
