package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingInterceptor logs incoming requests with method name and duration
func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	log.Printf("[gRPC] Method: %s, Request: %v", info.FullMethod, req)

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	log.Printf("[gRPC] Method: %s, Duration: %v ms, Error: %v", info.FullMethod, duration.Milliseconds(), err)

	return resp, err
}

// StreamLoggingInterceptor logs streaming requests
func StreamLoggingInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	log.Printf("[gRPC] Stream Method: %s, Is Client Stream: %v", info.FullMethod, info.IsClientStream)

	err := handler(srv, ss)

	duration := time.Since(start)
	log.Printf("[gRPC] Stream Method: %s, Duration: %v ms, Error: %v", info.FullMethod, duration.Milliseconds(), err)

	return err
}

