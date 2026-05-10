package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"order-service/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

const orderTTL = 5 * time.Minute

// OrderCache implements cache-aside pattern for orders.
type OrderCache struct {
	client *redis.Client
}

func NewOrderCache(client *redis.Client) *OrderCache {
	return &OrderCache{client: client}
}

func orderKey(id string) string {
	return fmt.Sprintf("order:%s", id)
}

// Get returns the cached order or nil if not found.
func (c *OrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	val, err := c.client.Get(ctx, orderKey(id)).Result()
	if err == redis.Nil {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}

	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}
	return &order, nil
}

// Set stores an order in cache with TTL.
func (c *OrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	return c.client.Set(ctx, orderKey(order.ID), data, orderTTL).Err()
}

// Invalidate deletes the cached order (call after status update).
func (c *OrderCache) Invalidate(ctx context.Context, id string) {
	if err := c.client.Del(ctx, orderKey(id)).Err(); err != nil {
		log.Printf("[Cache] Failed to invalidate order %s: %v", id, err)
	}
}
