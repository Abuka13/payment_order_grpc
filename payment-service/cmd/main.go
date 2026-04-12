package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"payment-service/internal/config"
	postgresrepo "payment-service/internal/repository/postgres"
	grpctransport "payment-service/internal/transport/grpc"
	httptransport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
	"payment-service/pb"

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
	dbURL := config.GetEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=Takanashi_13 dbname=paymentdb sslmode=disable")
	grpcPort := config.GetEnv("PAYMENT_GRPC_PORT", "9091")
	httpPort := config.GetEnv("PAYMENT_SERVICE_PORT", "8081")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	paymentRepo := postgresrepo.NewPaymentPostgresRepository(db)
	paymentUC := usecase.NewPaymentUsecase(paymentRepo)

	// Start gRPC server in goroutine
	go func() {
		grpcHandler := grpctransport.NewPaymentGRPCHandler(paymentUC)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port: %v", err)
		}

		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(grpctransport.LoggingInterceptor),
			grpc.StreamInterceptor(grpctransport.StreamLoggingInterceptor),
		)
		pb.RegisterPaymentServiceServer(grpcServer, grpcHandler)

		log.Printf("Payment Service gRPC server running on :%s\n", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Start HTTP server
	paymentHandler := httptransport.NewPaymentHandler(paymentUC)

	r := gin.Default()

	r.POST("/payments", paymentHandler.CreatePayment)
	r.GET("/payments/:order_id", paymentHandler.GetPaymentByOrderID)

	log.Printf("Payment Service HTTP running on :%s\n", httpPort)
	if err := r.Run(fmt.Sprintf(":%s", httpPort)); err != nil {
		log.Fatal(err)
	}
}
