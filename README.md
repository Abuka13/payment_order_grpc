# Assignment 4 — Production-Ready Scaling

This project extends Assignment 3 (**Orders + Payments + Notifications microservices**) by adding:

- Redis caching
- Reliable background workers
- Adapter pattern for external providers
- Redis-based idempotency
- API rate limiting

The system is built using:

- Go
- PostgreSQL
- RabbitMQ
- Redis
- Docker Compose

---

# Features

## 1. Redis Cache-Aside Pattern

The **Order Service** uses Redis as a cache layer.

### Request Flow

For `GET /orders/:id`:

1. Check Redis cache
2. If cache miss → fetch from PostgreSQL
3. Store result in Redis
4. Return response

### Cache TTL

- 5 minutes

### Cache Invalidation

Whenever an order changes status:

- paid
- failed
- cancelled

the corresponding Redis cache key is immediately deleted to avoid stale data.

---

## 2. Notification Background Worker

The Notification Service was redesigned into a proper asynchronous worker.

### EmailSender Interface

The system uses the Adapter Pattern through the `EmailSender` interface.

Available implementations:

- `SimulatedProvider`
- `SMTPProvider`

Provider selection is controlled via environment variables.

---

### Simulated Provider

The simulated provider:

- adds artificial network latency
- randomly fails 20% of requests
- helps test retry logic realistically

---

### SMTP Provider

The SMTP provider supports real email delivery through:

- Mailjet
- Gmail SMTP
- any standard SMTP server

---

## 3. Retry Logic

Failed email deliveries are retried using exponential backoff:

```text
2s → 4s → 8s → 16s → 32s
```

After maximum retries are exceeded, the message is moved to the:

- Dead Letter Queue (DLQ)

---

## 4. Redis-Based Idempotency

Duplicate RabbitMQ deliveries are prevented using Redis.

Before processing a notification, the worker checks:

```text
notif:processed:<event_id>
```

After successful delivery:

- the key is stored in Redis
- TTL = 24 hours

This guarantees duplicate events are ignored safely.

---

## 5. Bonus — API Rate Limiter

All HTTP routes in the Order Service are protected with a Redis-based rate limiter.

### Implementation

Uses Redis:

- `INCR`
- `EXPIRE`

### Default Limit

```text
10 requests per minute per IP
```

### Exceeded Limit Response

```http
429 Too Many Requests
```

Headers returned:

```text
X-RateLimit-Limit
X-RateLimit-Remaining
X-RateLimit-Reset
```

---

# Architecture

```text
Client
  └── Order Service (HTTP :8080)
        ├── Redis (cache + rate limiter)
        └── Payment Service (gRPC :9091)
              ├── PostgreSQL
              └── RabbitMQ
                    └── Notification Service
                          ├── Redis (idempotency)
                          └── EmailSender
                                ├── Simulated Provider
                                └── SMTP Provider
```

---

# Running the Project

Make sure Docker Desktop is running.

Start all services:

```bash
docker compose up --build
```

This starts:

- PostgreSQL
- Redis
- RabbitMQ
- Order Service
- Payment Service
- Notification Service

All services include health checks and proper startup ordering.

---

# Testing

## Create an Order

```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id":"c1",
    "item_name":"MacBook",
    "amount":150000,
    "customer_email":"test@example.com"
  }' | jq .
```

You should see notification logs inside Docker Compose output.

---

# Verify Cache-Aside Pattern

## First Request → Cache Miss

```bash
curl -s http://localhost:8080/orders/<ORDER_ID> | jq .
```

## Verify Redis Key Exists

```bash
docker exec -it redis redis-cli GET "order:<ORDER_ID>"
```

## Second Request → Cache Hit

```bash
curl -s http://localhost:8080/orders/<ORDER_ID> | jq .
```

---

# Verify Cache Invalidation

## Cancel an Order

```bash
curl -s -X PATCH \
  http://localhost:8080/orders/<ORDER_ID>/cancel | jq .
```

## Verify Key Was Removed

```bash
docker exec -it redis redis-cli EXISTS "order:<ORDER_ID>"
```

Expected output:

```text
(integer) 0
```

---

# Verify Idempotency Keys

```bash
docker exec -it redis redis-cli KEYS "notif:processed:*"
```

---

# Trigger Rate Limiter

```bash
for i in $(seq 1 12); do
  echo -n "req $i: "
  curl -s -o /dev/null -w "%{http_code}\n" \
    http://localhost:8080/orders/test
done
```

Expected:

```text
Requests 1-10  → 404
Requests 11-12 → 429
```

---

# RabbitMQ Management UI

URL:

```text
http://localhost:15672
```

Credentials:

```text
guest / guest
```

---

# Environment Variables

| Variable | Description | Default |
|---|---|---|
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `RATE_LIMIT` | Max requests per minute per IP | `10` |
| `PROVIDER_MODE` | Email provider mode | `SIMULATED` |
| `SIMULATED_FAILURE_RATE` | Random failure probability | `0.2` |

---

## SMTP Configuration

For the real SMTP provider configure:

```text
SMTP_HOST
SMTP_PORT
SMTP_USER
SMTP_PASS
SMTP_FROM
```

---

# Stopping the Project

```bash
docker compose down -v
```

The `-v` flag removes Docker volumes so the next startup begins with a clean database.
