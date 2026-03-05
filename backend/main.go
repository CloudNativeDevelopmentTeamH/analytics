package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"

	_ "github.com/joho/godotenv/autoload"

	"github.com/create-go-app/fiber-go-template/app/analytics"
	grpcapi "github.com/create-go-app/fiber-go-template/app/grpc"
	analyticsv1 "github.com/create-go-app/fiber-go-template/proto/analytics/v1"
)

func main() {
	httpAddr := envOr("HTTP_ADDR", ":8080") // health/ready
	grpcAddr := envOr("GRPC_ADDR", ":9090") // analytics read api

	// readiness: TODO: couple to kafka
	var ready atomic.Bool
	ready.Store(false)

	// Aggregator
	store := analytics.NewStore()

	// gRPC server
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	analyticsv1.RegisterAnalyticsServiceServer(grpcServer, grpcapi.NewAnalyticsServer(store))

	grpcErr := make(chan error, 1)
	go func() {
		log.Printf("gRPC listening on %s", grpcAddr)
		ready.Store(true)
		grpcErr <- grpcServer.Serve(grpcLis)
	}()

	// Fiber ops server (health/ready only)
	app := fiber.New()

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/readyz", func(c *fiber.Ctx) error {
		if ready.Load() {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.SendStatus(fiber.StatusServiceUnavailable)
	})

	httpErr := make(chan error, 1)
	go func() {
		log.Printf("HTTP (ops) listening on %s", httpAddr)
		httpErr <- app.Listen(httpAddr)
	}()

	// shutdown handling
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

	ready.Store(false)

	// stop fiber
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx
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

func envOr(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
