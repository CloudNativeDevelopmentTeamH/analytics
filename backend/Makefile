.PHONY: clean critic security lint test test.unit build run \
        proto proto.tools proto.gen \
        docker.build docker.up docker.down docker.logs docker.ps \
	kafka.topic.create kafka.produce.sample \
	db.migrate.up db.migrate.down

APP_NAME        := analytics-server
BUILD_DIR       := $(PWD)/build

# --- Proto ---
PROTO_DIR       := proto
PROTO_FILES     := $(shell find $(PROTO_DIR) -name "*.proto")
PROTOC          := protoc

# --- Docker/Compose (compose file is in repo root) ---
# If your docker-compose.yml is in repo root:
COMPOSE_FILE    := ../docker-compose.yml
COMPOSE         := docker compose -f $(COMPOSE_FILE)

# --- Kafka (Redpanda) ---
KAFKA_TOPIC     := focus.events

clean:
	rm -rf ./build
	find . -name "cover.out" -delete

critic:
	gocritic check -enableAll ./...

security:
	gosec ./...

lint:
	golangci-lint run ./...

test: clean critic security lint
	go test -v -timeout 30s -coverprofile=cover.out -cover ./...
	go tool cover -func=cover.out

test.unit:
	go test -v -timeout 30s ./...

build: test
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) main.go

run: build
	$(BUILD_DIR)/$(APP_NAME)

# -----------------------
# Protobuf generation
# -----------------------

proto.tools:
	@echo "Installing protoc Go plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto.gen:
	@echo "Generating Go code from proto..."
	$(PROTOC) -I $(PROTO_DIR) \
	  --go_out=. --go_opt=module=github.com/CloudNativeDevelopmentTeamH/analytics/backend \
	  --go-grpc_out=. --go-grpc_opt=module=github.com/CloudNativeDevelopmentTeamH/analytics/backend \
	  $(PROTO_FILES)

proto: proto.tools proto.gen

# -----------------------
# Docker / Compose
# -----------------------

docker.build:
	$(COMPOSE) build

docker.up:
	$(COMPOSE) up -d

docker.down:
	$(COMPOSE) down -v

docker.logs:
	$(COMPOSE) logs -f

docker.ps:
	$(COMPOSE) ps

# -----------------------
# Kafka helpers (Redpanda rpk)
# -----------------------

kafka.topic.create:
	$(COMPOSE) exec redpanda rpk topic create $(KAFKA_TOPIC) || true

kafka.produce.sample:
	printf '%s\n' '{"type":"FocusSessionEnded","sessionEnded":{"sessionId":"s1","userId":"u1","categoryId":"c1","durationSeconds":1500,"endedAt":"2026-03-05T12:00:00Z"}}' | $(COMPOSE) exec -T redpanda rpk topic produce $(KAFKA_TOPIC) -k test

# -----------------------
# Database migrations (Postgres)
# -----------------------

db.migrate.up:
	$(COMPOSE) exec -T postgres psql -U analytics -d analytics -v ON_ERROR_STOP=1 < platform/migrations/000001_create_init_tables.up.sql

db.migrate.down:
	$(COMPOSE) exec -T postgres psql -U analytics -d analytics -v ON_ERROR_STOP=1 < platform/migrations/000001_create_init_tables.down.sql
