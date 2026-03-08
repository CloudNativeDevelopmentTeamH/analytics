package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"

	_ "github.com/joho/godotenv/autoload"

	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/app/analytics"
	grpcapi "github.com/CloudNativeDevelopmentTeamH/analytics/backend/app/grpc"
	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/config"
	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/consumers"
	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/pkg/health"
	analyticsv1 "github.com/CloudNativeDevelopmentTeamH/analytics/backend/proto/analytics/v1"
)

func main() {
	// Load application.yml + ENV overrides (koanf)
	cfg := config.Load()

	httpAddr := ":" + strconv.Itoa(cfg.Server.HTTPPort) // health/ready
	grpcAddr := ":" + strconv.Itoa(cfg.Server.GRPCPort) // analytics read api

	// readiness = grpcReady && kafkaReady
	var ready atomic.Bool
	var grpcReady atomic.Bool
	var kafkaReady atomic.Bool

	updateReady := func() {
		ready.Store(grpcReady.Load() && kafkaReady.Load())
	}

	// Aggregator store (in-memory or postgres)
	store, closeStore, err := buildStore(cfg)
	if err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}
	defer func() {
		if closeStore != nil {
			_ = closeStore()
		}
	}()

	// Root context for background routines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- gRPC server ---
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	analyticsv1.RegisterAnalyticsServiceServer(grpcServer, grpcapi.NewAnalyticsServer(store))

	grpcErr := make(chan error, 1)
	go func() {
		log.Printf("gRPC listening on %s", grpcAddr)
		grpcReady.Store(true)
		updateReady()
		grpcErr <- grpcServer.Serve(grpcLis)
	}()

	// --- Kafka consumer ---
	consumer, err := consumers.NewKafkaConsumer(consumers.Config{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.GroupID,
	}, store, func(ok bool) {
		kafkaReady.Store(ok)
		updateReady()
	})
	if err != nil {
		log.Fatalf("failed to create kafka consumer: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	go consumer.Run(ctx)

	// --- Fiber ops server (health/ready only) ---
	app := fiber.New()
	health.RegisterRoutes(app, &ready)

	httpErr := make(chan error, 1)
	go func() {
		log.Printf("HTTP (ops) listening on %s", httpAddr)
		httpErr <- app.Listen(httpAddr)
	}()

	// --- shutdown handling ---
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sig:
		log.Printf("received signal: %s", s.String())
	case err := <-grpcErr:
		if err != nil {
			log.Printf("gRPC server stopped with error: %v", err)
		}
	case err := <-httpErr:
		if err != nil {
			log.Printf("HTTP server stopped with error: %v", err)
		}
	}

	// mark unready and stop background routines
	ready.Store(false)
	grpcReady.Store(false)
	kafkaReady.Store(false)
	cancel()

	// stop fiber
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	_ = ctxShutdown
	if err := app.Shutdown(); err != nil {
		log.Printf("fiber shutdown error: %v", err)
	}

	// stop grpc (gracefully, with timeout fallback)
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		log.Printf("gRPC graceful stop timeout -> forcing stop")
		grpcServer.Stop()
	}

	log.Println("shutdown complete")
}

func buildStore(cfg config.Config) (analytics.StatsStore, func() error, error) {
	switch cfg.Storage.Backend {
	case "in-memory":
		return analytics.NewStore(), nil, nil
	case "postgres":
		pgStore, err := analytics.NewPostgresStore(cfg.Storage.Postgres.DSN)
		if err != nil {
			return nil, nil, err
		}
		return pgStore, pgStore.Close, nil
	default:
		return nil, nil, fmt.Errorf("unsupported storage backend: %s", cfg.Storage.Backend)
	}
}
