package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/microcloud/bus"
	opsv1 "github.com/microcloud/gen/go/ops/v1"
	simv1 "github.com/microcloud/gen/go/sim/v1"
)

// StreamHub manages SSE connections for real-time updates
type StreamHub struct {
	subscriber *bus.Subscriber
	log        *slog.Logger

	mu      sync.RWMutex
	clients map[chan []byte]struct{}

	latestSnapshot *simv1.MetricSnapshot
	latestIncident *opsv1.Incident
	latestAction   *opsv1.Action
}

// NewStreamHub creates a new stream hub
func NewStreamHub(subscriber *bus.Subscriber, log *slog.Logger) *StreamHub {
	return &StreamHub{
		subscriber: subscriber,
		log:        log,
		clients:    make(map[chan []byte]struct{}),
	}
}

// Start begins listening to NATS subjects and broadcasting to clients
func (h *StreamHub) Start(ctx context.Context) error {
	// Subscribe to metrics
	metricsCC, err := h.subscriber.SubscribeMetrics(ctx, "orchestrator-metrics", func(ctx context.Context, snapshot *simv1.MetricSnapshot) error {
		h.mu.Lock()
		h.latestSnapshot = snapshot
		h.mu.Unlock()

		data, _ := json.Marshal(map[string]any{
			"type":    "metrics",
			"payload": snapshot,
		})
		h.broadcast(data)
		return nil
	})
	if err != nil {
		return fmt.Errorf("subscribe metrics: %w", err)
	}

	// Subscribe to incidents
	incidentsCC, err := h.subscriber.SubscribeIncidents(ctx, "orchestrator-incidents", func(ctx context.Context, incident *opsv1.Incident) error {
		h.mu.Lock()
		h.latestIncident = incident
		h.mu.Unlock()

		data, _ := json.Marshal(map[string]any{
			"type":    "incident",
			"payload": incident,
		})
		h.broadcast(data)
		return nil
	})
	if err != nil {
		metricsCC.Stop()
		return fmt.Errorf("subscribe incidents: %w", err)
	}

	// Subscribe to actions
	actionsCC, err := h.subscriber.SubscribeActions(ctx, "orchestrator-actions", func(ctx context.Context, action *opsv1.Action) error {
		h.mu.Lock()
		h.latestAction = action
		h.mu.Unlock()

		data, _ := json.Marshal(map[string]any{
			"type":    "action",
			"payload": action,
		})
		h.broadcast(data)
		return nil
	})
	if err != nil {
		metricsCC.Stop()
		incidentsCC.Stop()
		return fmt.Errorf("subscribe actions: %w", err)
	}

	h.log.Info("stream hub started")

	<-ctx.Done()
	metricsCC.Stop()
	incidentsCC.Stop()
	actionsCC.Stop()

	return ctx.Err()
}

func (h *StreamHub) broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- data:
		default:
			// Client too slow, skip
		}
	}
}

func (h *StreamHub) addClient(ch chan []byte) {
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
}

func (h *StreamHub) removeClient(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	close(ch)
	h.mu.Unlock()
}

// ServeHTTP handles SSE connections
func (h *StreamHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 100)
	h.addClient(ch)
	defer h.removeClient(ch)

	h.log.Debug("SSE client connected")

	// Send initial state
	h.mu.RLock()
	if h.latestSnapshot != nil {
		data, _ := json.Marshal(map[string]any{
			"type":    "metrics",
			"payload": h.latestSnapshot,
		})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	h.mu.RUnlock()

	// Keep-alive ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.log.Debug("SSE client disconnected")
			return
		case data := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
