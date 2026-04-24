package http

import (
	"database/sql"
	"errors"
	"net/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	uc *usecase.PaymentUsecase
}

func NewPaymentHandler(uc *usecase.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{uc: uc}
}

type createPaymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.uc.Create(req.OrderID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *PaymentHandler) GetPaymentByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")

	payment, err := h.uc.GetByOrderID(orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, payment)
}
