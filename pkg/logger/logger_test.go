package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{
		Level:       "info",
		Format:      "json",
		ServiceName: "test-service",
		Output:      &buf,
	})

	log.Info("test message", "key", "value")

	out := buf.String()
	if !strings.Contains(out, `"msg":"test message"`) {
		t.Errorf("expected JSON msg field, got: %s", out)
	}
	if !strings.Contains(out, `"service":"test-service"`) {
		t.Errorf("expected service field, got: %s", out)
	}
}

func TestWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := New(Config{Format: "json", Output: &buf})

	ctx := WithRequestID(context.Background(), "req-123")
	FromContext(ctx, log).Info("with request")

	if !strings.Contains(buf.String(), `"request_id":"req-123"`) {
		t.Errorf("expected request_id in output: %s", buf.String())
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"},
	}
	for _, tt := range tests {
		got := parseLevel(tt.input).String()
		if got != tt.want {
			t.Errorf("parseLevel(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
