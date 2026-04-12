package main

import (
	"database/sql"
	"log"
	postgresrepo "payment-service/internal/repository/postgres"
	httptransport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=Takanashi_13 dbname=paymentdb sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	paymentRepo := postgresrepo.NewPaymentPostgresRepository(db)
	paymentUC := usecase.NewPaymentUsecase(paymentRepo)
	paymentHandler := httptransport.NewPaymentHandler(paymentUC)

	r := gin.Default()

	r.POST("/payments", paymentHandler.CreatePayment)
	r.GET("/payments/:order_id", paymentHandler.GetPaymentByOrderID)

	log.Println("Payment Service running on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}
