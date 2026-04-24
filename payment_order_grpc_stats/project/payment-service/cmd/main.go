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
	if _, err := os.Stat(".env"); err == nil {
		loadEnvFile(".env")
	}
}

func loadEnvFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	content := string(data)
	start := 0
	for i := 0; i <= len(content); i++ {
		if i == len(content) || content[i] == '\n' {
			line := content[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if len(line) > 0 && line[0] != '#' {
				for j, ch := range line {
					if ch == '=' {
						key := line[:j]
						val := line[j+1:]
						if os.Getenv(key) == "" {
							os.Setenv(key, val)
						}
						break
					}
				}
			}
			start = i + 1
		}
	}
}

func main() {
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
	log.Println("Payment Service connected to database")

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
