package main

import (
	"database/sql"
	"log"
	"net/http"
	"order-service/internal/client"
	postgresrepo "order-service/internal/repository/postgres"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=Takanashi_13 dbname=orderdb sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	orderRepo := postgresrepo.NewOrderPostgresRepository(db)
	paymentClient := client.NewPaymentHTTPClient("http://localhost:8081", httpClient)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	orderHandler := httptransport.NewOrderHandler(orderUC)

	r := gin.Default()

	r.POST("/orders", orderHandler.CreateOrder)
	r.GET("/orders/:id", orderHandler.GetOrder)
	r.PATCH("/orders/:id/cancel", orderHandler.CancelOrder)

	log.Println("Order Service running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
