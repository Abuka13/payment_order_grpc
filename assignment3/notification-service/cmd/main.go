package main

import (
	"database/sql"
	"log"
	"notification-service/internal/broker"
	"notification-service/internal/consumer"
	"notification-service/internal/idempotency"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	dbURL := getEnv("DATABASE_URL", "host=localhost port=5432 user=postgres password=Takanashi_13 dbname=notificationdb sslmode=disable")

	// --- Connect to PostgreSQL for idempotency store ---
	db, err := connectDB(dbURL)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to DB: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("[Main] Migration failed: %v", err)
	}

	idemStore := idempotency.NewStore(db)

	// --- Connect to RabbitMQ ---
	conn, err := broker.Connect(rabbitURL)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	notifConsumer := consumer.NewNotificationConsumer(conn.Channel, idemStore)

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

func connectDB(url string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", url)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				log.Println("[Main] Connected to notificationdb")
				return db, nil
			}
		}
		log.Printf("[Main] Waiting for DB... attempt %d", i+1)
		time.Sleep(3 * time.Second)
	}
	return nil, err
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS processed_events (
			event_id TEXT PRIMARY KEY,
			processed_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
