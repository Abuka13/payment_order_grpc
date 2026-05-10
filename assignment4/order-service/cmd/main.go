package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"order-service/internal/cache"
	"order-service/internal/client"
	"order-service/internal/config"
	"order-service/internal/middleware"
	postgresrepo "order-service/internal/repository/postgres"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/pb"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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
	dbURL := config.GetEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=Takanashi_13 dbname=orderdb sslmode=disable")
	paymentGRPCAddr := config.GetEnv("PAYMENT_GRPC_ADDRESS", "localhost:9091")
	orderGRPCPort := config.GetEnv("ORDER_GRPC_PORT", "9090")
	orderHTTPPort := config.GetEnv("ORDER_SERVICE_PORT", "8080")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

	// Rate limiter config (defaults: 10 req / 1 min)
	rateLimitStr := config.GetEnv("RATE_LIMIT", "10")
	rateLimit, _ := strconv.Atoi(rateLimitStr)
	if rateLimit <= 0 {
		rateLimit = 10
	}

	// --- Database ---
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("Order Service connected to database")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	orderCache := cache.NewOrderCache(rdb)
	log.Printf("Order Service connected to Redis at %s", redisAddr)

	// --- Payment gRPC client ---
	paymentClient, err := client.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to connect to payment service: %v", err)
	}
	defer paymentClient.Close()

	// --- Use case & repository ---
	orderRepo := postgresrepo.NewOrderPostgresRepository(db)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient, orderCache)

	// --- Graceful shutdown ---
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	var grpcServer *grpc.Server

	go func() {
		grpcHandler := grpctransport.NewOrderGRPCHandler(orderUC, orderRepo)
		listener, err := net.Listen("tcp", fmt.Sprintf(":%s", orderGRPCPort))
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port: %v", err)
		}
		grpcServer = grpc.NewServer()
		pb.RegisterOrderServiceServer(grpcServer, grpcHandler)
		log.Printf("Order Service gRPC server running on :%s\n", orderGRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	go func() {
		orderHandler := httptransport.NewOrderHandler(orderUC)
		r := gin.Default()

		// Bonus: rate-limiter middleware (10 req/min per IP, backed by Redis)
		r.Use(middleware.RateLimiter(rdb, rateLimit, time.Minute))

		r.POST("/orders", orderHandler.CreateOrder)
		r.GET("/orders/:id", orderHandler.GetOrder)
		r.PATCH("/orders/:id/cancel", orderHandler.CancelOrder)
		log.Printf("Order Service HTTP running on :%s\n", orderHTTPPort)
		if err := r.Run(fmt.Sprintf(":%s", orderHTTPPort)); err != nil {
			log.Fatal(err)
		}
	}()

	sig := <-stopCh
	log.Printf("Order Service received signal: %v — shutting down gracefully", sig)
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}
	db.Close()
	rdb.Close()
	log.Println("Order Service stopped")
}
