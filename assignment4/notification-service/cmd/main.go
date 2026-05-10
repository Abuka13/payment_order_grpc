package main

import (
	"log"
	"notification-service/internal/broker"
	"notification-service/internal/consumer"
	"notification-service/internal/provider"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	providerMode := getEnv("PROVIDER_MODE", "SIMULATED")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	log.Printf("[Main] Connected to Redis at %s", redisAddr)

	// --- Email provider (Adapter Pattern) ---
	var sender provider.EmailSender
	switch providerMode {
	case "REAL":
		sender = provider.NewSMTPProvider(
			getEnv("SMTP_HOST", "smtp.example.com"),
			getEnv("SMTP_PORT", "587"),
			getEnv("SMTP_USER", ""),
			getEnv("SMTP_PASS", ""),
			getEnv("SMTP_FROM", "no-reply@example.com"),
		)
		log.Println("[Main] Using REAL SMTP provider")
	default:
		failureRate, _ := strconv.ParseFloat(getEnv("SIMULATED_FAILURE_RATE", "0.2"), 64)
		sender = provider.NewSimulatedProvider(failureRate)
		log.Printf("[Main] Using SIMULATED provider (failure rate: %.0f%%)", failureRate*100)
	}

	// --- RabbitMQ ---
	conn, err := broker.Connect(rabbitURL)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	notifConsumer := consumer.NewNotificationConsumer(conn.Channel, rdb, sender)

	// --- Graceful shutdown ---
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("[Main] Received signal: %v — initiating graceful shutdown", sig)
		close(stopCh)
	}()

	log.Println("[Main] Notification Service started")
	if err := notifConsumer.Start(stopCh); err != nil {
		log.Fatalf("[Main] Consumer error: %v", err)
	}
	log.Println("[Main] Notification Service stopped gracefully")
}

func connectWithRetry(fn func() error, maxAttempts int) error {
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
