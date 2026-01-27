// Package logger provides structured logging for microcloud services.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	serviceKey   contextKey = "service"
)

// Config holds logger configuration
type Config struct {
	Level       string // "debug", "info", "warn", "error"
	Format      string // "json" or "text"
	ServiceName string
	Output      io.Writer // defaults to os.Stdout
}

// New creates a new configured slog.Logger
func New(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(cfg.Output, opts)
	} else {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	if cfg.ServiceName != "" {
		handler = &serviceHandler{
			Handler:     handler,
			serviceName: cfg.ServiceName,
		}
	}

	return slog.New(handler)
}

// NewFromEnv creates a logger from environment variables:
// LOG_LEVEL (default: info), LOG_FORMAT (default: json), SERVICE_NAME
func NewFromEnv(serviceName string) *slog.Logger {
	return New(Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "json"),
		ServiceName: serviceName,
	})
}

// WithRequestID adds a request ID to the context for logging
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// FromContext extracts logger fields from context and returns enriched logger
func FromContext(ctx context.Context, log *slog.Logger) *slog.Logger {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		log = log.With("request_id", reqID)
	}
	return log
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// serviceHandler wraps a handler to add service name to all records
type serviceHandler struct {
	slog.Handler
	serviceName string
}

func (h *serviceHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.String("service", h.serviceName))
	return h.Handler.Handle(ctx, r)
}

func (h *serviceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &serviceHandler{Handler: h.Handler.WithAttrs(attrs), serviceName: h.serviceName}
}

func (h *serviceHandler) WithGroup(name string) slog.Handler {
	return &serviceHandler{Handler: h.Handler.WithGroup(name), serviceName: h.serviceName}
}
