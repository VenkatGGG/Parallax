package bus

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.URL != "nats://localhost:4222" {
		t.Errorf("unexpected URL: %s", cfg.URL)
	}
	if cfg.StreamName != "MICROCLOUD" {
		t.Errorf("unexpected stream name: %s", cfg.StreamName)
	}
	if cfg.MaxReconnects != -1 {
		t.Errorf("unexpected max reconnects: %d", cfg.MaxReconnects)
	}
}

func TestSubjectConstants(t *testing.T) {
	tests := []struct {
		name   string
		got    string
		want   string
	}{
		{"SimMetrics", SubjectSimMetrics, "sim.metrics"},
		{"SimEvents", SubjectSimEvents, "sim.events"},
		{"OpsIncidents", SubjectOpsIncidents, "ops.incidents"},
		{"OpsActions", SubjectOpsActions, "ops.actions"},
		{"OpsCommands", SubjectOpsCommands, "ops.commands"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %s, want %s", tt.name, tt.got, tt.want)
		}
	}
}
