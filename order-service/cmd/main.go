package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"order-service/internal/client"
	"order-service/internal/config"
	postgresrepo "order-service/internal/repository/postgres"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/pb"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func init() {
	// Load .env file if exists
	if _, err := os.Stat(".env"); err == nil {
		loadEnvFile(".env")
	}
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			content := string(buf[:n])
			// Simple env parsing
		}
		if err != nil {
			break
		}
	}
}

func main() {
	// Get configuration from environment variables
	dbURL := config.GetEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=Takanashi_13 dbname=orderdb sslmode=disable")
	paymentGRPCAddr := config.GetEnv("PAYMENT_GRPC_ADDRESS", "localhost:9091")
	orderGRPCPort := config.GetEnv("ORDER_GRPC_PORT", "9090")
	orderHTTPPort := config.GetEnv("ORDER_SERVICE_PORT", "8080")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Initialize gRPC payment client
	paymentClient, err := client.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to connect to payment service: %v", err)
	}
	defer paymentClient.Close()

	orderRepo := postgresrepo.NewOrderPostgresRepository(db)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	orderHandler := httptransport.NewOrderHandler(orderUC)

	// Start gRPC server in goroutine for order streaming
	go func() {
		grpcHandler := grpctransport.NewOrderGRPCHandler(orderUC, orderRepo)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%s", orderGRPCPort))
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port: %v", err)
		}

		grpcServer := grpc.NewServer()
		pb.RegisterOrderServiceServer(grpcServer, grpcHandler)

		log.Printf("Order Service gRPC server running on :%s\n", orderGRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Start HTTP server
	r := gin.Default()

	r.POST("/orders", orderHandler.CreateOrder)
	r.GET("/orders/:id", orderHandler.GetOrder)
	r.PATCH("/orders/:id/cancel", orderHandler.CancelOrder)

	log.Printf("Order Service HTTP running on :%s\n", orderHTTPPort)
	if err := r.Run(fmt.Sprintf(":%s", orderHTTPPort)); err != nil {
		log.Fatal(err)
	}
}
