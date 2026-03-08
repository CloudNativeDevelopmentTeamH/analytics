package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

func Load() Config {
	k := koanf.New(".")

	// --- defaults ---
	k.Set("server.http_port", 8080)
	k.Set("server.grpc_port", 9090)

	k.Set("kafka.brokers", []string{"localhost:9092"})
	k.Set("kafka.topic", "focus.events")
	k.Set("kafka.group_id", "analytics-service")

	k.Set("storage.backend", "in-memory")
	k.Set("storage.postgres.dsn", "postgres://analytics:analytics@localhost:5432/analytics?sslmode=disable")

	// --- YAML ---
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config/application.yml"
	}

	if _, err := os.Stat(cfgPath); err == nil {
		if err := k.Load(file.Provider(cfgPath), yaml.Parser()); err != nil {
			log.Fatalf("failed to load config file: %v", err)
		}
	}

	// --- ENV overrides ---
	if err := k.Load(env.Provider("", ".", func(s string) string {
		return s
	}), nil); err != nil {
		log.Fatalf("failed loading env: %v", err)
	}

	applyEnvOverrides(k)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		log.Fatalf("config unmarshal failed: %v", err)
	}

	validate(cfg)

	return cfg
}

func applyEnvOverrides(k *koanf.Koanf) {

	if v := os.Getenv("HTTP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			k.Set("server.http_port", n)
		}
	}

	if v := os.Getenv("GRPC_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			k.Set("server.grpc_port", n)
		}
	}

	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		parts := strings.Split(v, ",")
		var brokers []string

		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				brokers = append(brokers, p)
			}
		}

		if len(brokers) > 0 {
			k.Set("kafka.brokers", brokers)
		}
	}

	if v := os.Getenv("KAFKA_TOPIC"); v != "" {
		k.Set("kafka.topic", v)
	}

	if v := os.Getenv("KAFKA_GROUP_ID"); v != "" {
		k.Set("kafka.group_id", v)
	}

	if v := os.Getenv("STORE_BACKEND"); v != "" {
		k.Set("storage.backend", v)
	}

	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		k.Set("storage.postgres.dsn", v)
	}
}

func validate(cfg Config) {

	if cfg.Server.HTTPPort <= 0 {
		log.Fatalf("invalid http port")
	}

	if cfg.Server.GRPCPort <= 0 {
		log.Fatalf("invalid grpc port")
	}

	if len(cfg.Kafka.Brokers) == 0 {
		log.Fatalf("kafka brokers must not be empty")
	}

	if cfg.Kafka.Topic == "" {
		log.Fatalf("kafka topic must not be empty")
	}

	if cfg.Kafka.GroupID == "" {
		log.Fatalf("kafka group_id must not be empty")
	}

	if cfg.Storage.Backend == "" {
		log.Fatalf("storage backend must not be empty")
	}

	if cfg.Storage.Backend == "postgres" && cfg.Storage.Postgres.DSN == "" {
		log.Fatalf("storage postgres dsn must not be empty when backend=postgres")
	}
}
