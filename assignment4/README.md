# Assignment 4 — Production-Ready Scaling

## What's New (vs Assignment 3)

| Feature | Where |
|---|---|
| Redis Cache-Aside (GET /orders/:id) | `order-service/internal/cache/cache.go` |
| Atomic Cache Invalidation (on status update) | `order-service/internal/usecase/order_usecase.go` |
| Rate Limiter middleware (bonus) | `order-service/internal/middleware/rate_limiter.go` |
| EmailSender interface (Adapter Pattern) | `notification-service/internal/provider/interface.go` |
| Simulated provider (latency + random failures) | `notification-service/internal/provider/simulated.go` |
| Real SMTP provider adapter | `notification-service/internal/provider/smtp.go` |
| Redis-backed Idempotency (replaces Postgres table) | `notification-service/internal/consumer/consumer.go` |
| Exponential Backoff retry (2s→4s→8s→16s→32s) | `notification-service/internal/consumer/consumer.go` |
| Redis container | `docker-compose.yml` |

---

## Cache Invalidation Strategy

The Order Service uses the **cache-aside** pattern:

- **Read path**: `GET /orders/:id` checks Redis first (TTL = 5 min). On a cache miss, the DB result is written back to Redis for subsequent reads.
- **Write path** (status change): After every `UpdateStatus` call in the DB, `cache.Invalidate()` is called **immediately and atomically** in the same use-case method. This means the next read always fetches a fresh value from the DB and re-populates the cache — there is no window where a stale "Pending" is served for a paid order.

```
CreateOrder / Cancel
    → DB.UpdateStatus(...)   ← source of truth updated
    → cache.Invalidate(id)  ← cache entry deleted immediately
    → next GET hits DB, re-populates cache
```

---

## Retry Logic (Exponential Backoff)

The notification worker retries failed provider calls with exponential backoff before giving up and routing the message to the Dead-Letter Queue.

```
Attempt 1  → immediate
Attempt 2  → wait 2s
Attempt 3  → wait 4s
Attempt 4  → wait 8s
Attempt 5  → wait 16s
           → NACK → DLQ
```

Formula: `delay = 2^attempt × 2s`

---

## Idempotency

Before sending a notification the worker checks a Redis key:
```
notif:processed:<event_id>
```
If the key exists, the message is ACKed without resending. After a successful send, the key is set with a **24-hour TTL** (long enough to cover any retry storm, short enough to self-clean).

This prevents duplicate emails even when:
- The consumer crashes after sending but before ACKing.
- RabbitMQ redelivers the message after a network hiccup.

---

## Provider Selection

Set `PROVIDER_MODE` in the environment:

| Value | Behaviour |
|---|---|
| `SIMULATED` (default) | Logs the action, sleeps 50–200 ms, fails randomly at `SIMULATED_FAILURE_RATE` (default 20%) |
| `REAL` | Sends via SMTP using `SMTP_HOST/PORT/USER/PASS/FROM` env vars |

---

## Bonus — Rate Limiter

Every route in the Order Service HTTP server is protected by a Redis-backed rate limiter:
- **Limit**: configurable via `RATE_LIMIT` env var (default: 10 requests / minute per IP).
- **Storage**: Redis `INCR` + `EXPIRE` — works correctly across multiple service replicas.
- **Response on breach**: HTTP `429 Too Many Requests` with `Retry-After` seconds.
- **Headers returned**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

---

## Running

```bash
docker compose up --build
```

### Quick test

```bash
# Create an order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","item_name":"Book","amount":1999,"customer_email":"user@example.com"}'

# Fetch (first call → DB + cache write, second call → cache hit)
curl http://localhost:8080/orders/<id>

# Rate limiter — fire 11 requests in quick succession
for i in $(seq 1 11); do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/orders/fake; done
```
