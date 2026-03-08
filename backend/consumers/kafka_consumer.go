package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/app/analytics"
	"github.com/segmentio/kafka-go"
)

// NOTE: Focus service has no Kafka yet, so we define a minimal, flexible event envelope.
// You can evolve this later without changing the consumer loop shape.

const (
	EventTypeSessionEnded = "FocusSessionEnded"
)

// FocusEventEnvelope is a generic event wrapper for a single topic approach (recommended for MVP).
type FocusEventEnvelope struct {
	Type string `json:"type"`

	// SessionEnded payload (optional depending on Type)
	SessionEnded *FocusSessionEnded `json:"sessionEnded,omitempty"`
}

// FocusSessionEnded is the only event analytics needs for now.
type FocusSessionEnded struct {
	SessionID       string `json:"sessionId"`
	UserID          string `json:"userId"`
	CategoryID      string `json:"categoryId,omitempty"` // empty => uncategorized
	DurationSeconds int64  `json:"durationSeconds"`
	EndedAt         string `json:"endedAt,omitempty"` // ISO-8601 string (optional for now)
}

// Consumer consumes focus events and updates the in-memory analytics store.
type Consumer struct {
	reader *kafka.Reader
	store  analytics.StatsStore

	// ready becomes true once we successfully connected & started consuming.
	// You can wire this to /readyz in main.go.
	ready func(bool)
}

type Config struct {
	Brokers []string
	Topic   string
	GroupID string

	// Tuning
	MinBytes int
	MaxBytes int

	// Commit/Read settings
	MaxWait time.Duration
}

func NewKafkaConsumer(cfg Config, store analytics.StatsStore, readyFn func(bool)) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka brokers must not be empty")
	}
	if cfg.Topic == "" {
		return nil, errors.New("kafka topic must not be empty")
	}
	if cfg.GroupID == "" {
		return nil, errors.New("kafka groupID must not be empty")
	}
	if store == nil {
		return nil, errors.New("store must not be nil")
	}

	minBytes := cfg.MinBytes
	maxBytes := cfg.MaxBytes
	if minBytes == 0 {
		minBytes = 1e3 // 1KB
	}
	if maxBytes == 0 {
		maxBytes = 10e6 // 10MB
	}

	rc := kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: minBytes,
		MaxBytes: maxBytes,
	}

	if cfg.MaxWait > 0 {
		rc.MaxWait = cfg.MaxWait
	}

	r := kafka.NewReader(rc)

	c := &Consumer{
		reader: r,
		store:  store,
		ready:  readyFn,
	}
	return c, nil
}

func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// Run blocks until ctx is cancelled. It reads events and updates the store.
// IMPORTANT:
// - With GroupID, kafka-go commits offsets automatically after ReadMessage returns.
// - If you want "process-then-commit", switch to FetchMessage + CommitMessages.
func (c *Consumer) Run(ctx context.Context) {
	if c.ready != nil {
		c.ready(true)
	}

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// ctx cancelled => normal shutdown
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka ReadMessage error: %v", err)
			continue
		}

		if err := c.handleMessage(ctx, msg); err != nil {
			// For MVP: log and continue.
			// Later you might want DLQ or metrics.
			log.Printf("kafka handleMessage error: %v (topic=%s partition=%d offset=%d)",
				err, msg.Topic, msg.Partition, msg.Offset)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	// Decode JSON event
	var env FocusEventEnvelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		return err
	}

	switch env.Type {
	case EventTypeSessionEnded:
		// Support both envelope.SessionEnded and a "flat" message if you decide so later.
		if env.SessionEnded == nil {
			// Try parse as flat payload as fallback:
			var flat FocusSessionEnded
			if err := json.Unmarshal(msg.Value, &flat); err != nil {
				return errors.New("sessionEnded payload missing and flat parse failed")
			}
			return c.applySessionEnded(flat)
		}
		return c.applySessionEnded(*env.SessionEnded)

	default:
		// Unknown event type: ignore.
		return nil
	}
}

func (c *Consumer) applySessionEnded(e FocusSessionEnded) error {
	if e.UserID == "" {
		return errors.New("missing userId")
	}
	if e.DurationSeconds <= 0 {
		return errors.New("durationSeconds must be > 0")
	}

	// categoryId can be empty => store will treat as Uncategorized
	return c.store.IngestSessionEnded(e.UserID, e.CategoryID, e.DurationSeconds)
}
