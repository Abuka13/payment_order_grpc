package http

import (
	"database/sql"
	"errors"
	"net/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	uc *usecase.OrderUsecase
}

func NewOrderHandler(uc *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id"`
	ItemName   string `json:"item_name"`
	Amount     int64  `json:"amount"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.uc.Create(req.CustomerID, req.ItemName, req.Amount)
	if err != nil {
		if errors.Is(err, contextDeadlineExceededLike()) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service unavailable"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.uc.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.uc.Cancel(id)
	if err != nil {
		if err.Error() == "only pending orders can be cancelled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}

func contextDeadlineExceededLike() error {
	return errors.New("deadline exceeded")
}
