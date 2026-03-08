package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_UsesYAMLValues(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	t.Setenv("GRPC_PORT", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("KAFKA_TOPIC", "")
	t.Setenv("KAFKA_GROUP_ID", "")
	t.Setenv("STORE_BACKEND", "")
	t.Setenv("POSTGRES_DSN", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "application.yml")

	content := `server:
  http_port: 18080
  grpc_port: 19090

kafka:
  brokers:
    - broker-a:9092
  topic: analytics.events
  group_id: analytics-tests

storage:
  backend: postgres
  postgres:
    dsn: postgres://u:p@localhost:5432/db?sslmode=disable
`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	t.Setenv("CONFIG_PATH", path)

	cfg := Load()

	if cfg.Server.HTTPPort != 18080 {
		t.Fatalf("expected http port 18080, got %d", cfg.Server.HTTPPort)
	}
	if cfg.Server.GRPCPort != 19090 {
		t.Fatalf("expected grpc port 19090, got %d", cfg.Server.GRPCPort)
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "broker-a:9092" {
		t.Fatalf("unexpected brokers: %+v", cfg.Kafka.Brokers)
	}
	if cfg.Kafka.Topic != "analytics.events" {
		t.Fatalf("expected topic analytics.events, got %s", cfg.Kafka.Topic)
	}
	if cfg.Kafka.GroupID != "analytics-tests" {
		t.Fatalf("expected group id analytics-tests, got %s", cfg.Kafka.GroupID)
	}
	if cfg.Storage.Backend != "postgres" {
		t.Fatalf("expected backend postgres, got %s", cfg.Storage.Backend)
	}
	if cfg.Storage.Postgres.DSN != "postgres://u:p@localhost:5432/db?sslmode=disable" {
		t.Fatalf("unexpected dsn: %s", cfg.Storage.Postgres.DSN)
	}
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.yml")

	content := `server:
  http_port: 8080
  grpc_port: 9090

kafka:
  brokers:
    - from-yaml:9092
  topic: yaml.topic
  group_id: yaml-group

storage:
  backend: in-memory
  postgres:
    dsn: postgres://yaml
`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	t.Setenv("CONFIG_PATH", path)
	t.Setenv("HTTP_PORT", "8181")
	t.Setenv("GRPC_PORT", "9191")
	t.Setenv("KAFKA_BROKERS", "a:9092, b:9092")
	t.Setenv("KAFKA_TOPIC", "env.topic")
	t.Setenv("KAFKA_GROUP_ID", "env-group")
	t.Setenv("STORE_BACKEND", "postgres")
	t.Setenv("POSTGRES_DSN", "postgres://env")

	cfg := Load()

	if cfg.Server.HTTPPort != 8181 {
		t.Fatalf("expected http port 8181, got %d", cfg.Server.HTTPPort)
	}
	if cfg.Server.GRPCPort != 9191 {
		t.Fatalf("expected grpc port 9191, got %d", cfg.Server.GRPCPort)
	}
	if len(cfg.Kafka.Brokers) != 2 || cfg.Kafka.Brokers[0] != "a:9092" || cfg.Kafka.Brokers[1] != "b:9092" {
		t.Fatalf("unexpected brokers: %+v", cfg.Kafka.Brokers)
	}
	if cfg.Kafka.Topic != "env.topic" {
		t.Fatalf("expected topic env.topic, got %s", cfg.Kafka.Topic)
	}
	if cfg.Kafka.GroupID != "env-group" {
		t.Fatalf("expected group id env-group, got %s", cfg.Kafka.GroupID)
	}
	if cfg.Storage.Backend != "postgres" {
		t.Fatalf("expected backend postgres, got %s", cfg.Storage.Backend)
	}
	if cfg.Storage.Postgres.DSN != "postgres://env" {
		t.Fatalf("unexpected dsn: %s", cfg.Storage.Postgres.DSN)
	}
}
