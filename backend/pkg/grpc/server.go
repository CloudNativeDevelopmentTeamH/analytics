package grpc

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

// Run starts a gRPC server on the given address (e.g. ":9090").
// Pass a register callback to register all service implementations.
func Run(addr string, register func(s *grpc.Server)) (*grpc.Server, net.Listener, error) {
	if addr == "" {
		addr = ":9090"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc listen on %s: %w", addr, err)
	}

	s := grpc.NewServer()
	if register != nil {
		register(s)
	}

	log.Printf("gRPC server listening on %s", addr)
	go func() {
		if err := s.Serve(lis); err != nil {
			// Serve returns on Stop/GracefulStop too; caller decides how to handle lifecycle.
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	return s, lis, nil
}
