package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)


func RateLimiter(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		ip := c.ClientIP()
		key := fmt.Sprintf("rate:%s", ip)

		// INCR atomically increments the counter; returns new value
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Redis unavailable — fail open to avoid blocking legit traffic
			c.Next()
			return
		}

		// On first request, set the expiry for the window
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		// Set headers so clients can self-throttle
		ttl, _ := rdb.TTL(ctx, key).Result()
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, limit-int(count))))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

		if int(count) > limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"limit":       limit,
				"window":      window.String(),
				"retry_after": ttl.Seconds(),
			})
			return
		}

		c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
