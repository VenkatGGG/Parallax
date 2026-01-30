package storage

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Host != "localhost" {
		t.Errorf("unexpected host: %s", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("unexpected port: %d", cfg.Port)
	}
	if cfg.Database != "microcloud" {
		t.Errorf("unexpected database: %s", cfg.Database)
	}
	if cfg.User != "microcloud" {
		t.Errorf("unexpected user: %s", cfg.User)
	}
}

func TestConfigDSN(t *testing.T) {
	cfg := DefaultConfig()
	dsn := cfg.DSN()
	expected := "postgres://microcloud:microcloud@localhost:5432/microcloud?sslmode=disable"
	if dsn != expected {
		t.Errorf("unexpected DSN:\ngot:  %s\nwant: %s", dsn, expected)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("DB_HOST", "testhost")
	t.Setenv("DB_PORT", "5555")
	t.Setenv("DB_NAME", "testdb")

	cfg := ConfigFromEnv()
	if cfg.Host != "testhost" {
		t.Errorf("unexpected host: %s", cfg.Host)
	}
	if cfg.Port != 5555 {
		t.Errorf("unexpected port: %d", cfg.Port)
	}
	if cfg.Database != "testdb" {
		t.Errorf("unexpected database: %s", cfg.Database)
	}
}
