Assignment 4 — Production-Ready Scaling
This project extends Assignment 3 (Orders + Payments + Notifications microservices) with Redis caching, a reliable background worker, the Adapter pattern for external providers, and a bonus API rate limiter.
The stack is Go, PostgreSQL, RabbitMQ, and Redis — all wired together via Docker Compose.

What was added
Order Service now sits in front of Redis. Every GET /orders/:id checks the cache before touching the database. If there's a miss, the result gets written back with a 5-minute TTL. Whenever an order's status changes — paid, failed, cancelled — the cache key is deleted immediately in the same use-case call, so stale data never leaks through.
Notification Service was refactored into a proper background worker. The sendNotification inline function is gone; in its place is an EmailSender interface with two implementations you can swap via an environment variable. The simulated provider adds realistic network latency and randomly fails 20% of the time so retry logic actually gets exercised. The real provider sends via SMTP and is ready to point at Mailjet or any standard mail server.
If a send fails, the worker retries with exponential backoff — 2s, 4s, 8s, 16s, 32s — before giving up and routing the message to the dead-letter queue. Idempotency is now handled through Redis instead of a Postgres table: before processing, the worker checks for a notif:processed:<event_id> key. After a successful send it writes that key with a 24-hour TTL, so duplicate deliveries from RabbitMQ are silently discarded.
Bonus — Rate Limiter. A middleware wraps all HTTP routes in the Order Service. It uses Redis INCR + EXPIRE to count requests per client IP within a sliding one-minute window. Exceeding the limit (default: 10 req/min) returns 429 Too Many Requests along with X-RateLimit-Limit, X-RateLimit-Remaining, and X-RateLimit-Reset headers.

Architecture
Client
  └── Order Service (HTTP :8080)
        ├── Redis (cache-aside, rate limiter)
        └── Payment Service (gRPC :9091)
              ├── PostgreSQL
              └── RabbitMQ → Notification Service
                                ├── Redis (idempotency)
                                └── EmailSender (Simulated | SMTP)

Running
Make sure Docker Desktop is running, then:
bashdocker compose up --build
That's it. Postgres, Redis, RabbitMQ, and all three services start up with healthchecks in the right order.

Testing
Create an order and watch the notification log:
bashcurl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","item_name":"MacBook","amount":150000,"customer_email":"test@example.com"}' | jq .
In the compose log you'll see the simulated provider log the email, or retry attempts if it randomly failed.
Verify the cache-aside pattern:
bash# First call — cache miss, goes to DB, writes to Redis
curl -s http://localhost:8080/orders/<ORDER_ID> | jq .

# Check the key is now in Redis
docker exec -it redis redis-cli GET "order:<ORDER_ID>"

# Second call — served from Redis
curl -s http://localhost:8080/orders/<ORDER_ID> | jq .
Verify cache invalidation:
bash# Cancel the order
curl -s -X PATCH http://localhost:8080/orders/<ORDER_ID>/cancel | jq .

# Key should be gone
docker exec -it redis redis-cli EXISTS "order:<ORDER_ID>"
# → (integer) 0
Verify idempotency keys:
bashdocker exec -it redis redis-cli KEYS "notif:processed:*"
Trigger the rate limiter:
bashfor i in $(seq 1 12); do
  echo -n "req $i: "
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/orders/test
done
# Requests 1-10 → 404, requests 11-12 → 429
RabbitMQ management UI: http://localhost:15672 (guest / guest)

Configuration
All config lives in environment variables. The most relevant ones:
REDIS_ADDR — Redis address, default localhost:6379
RATE_LIMIT — max requests per minute per IP for the Order Service, default 10
PROVIDER_MODE — set to SIMULATED (default) or REAL in the Notification Service
SIMULATED_FAILURE_RATE — float between 0 and 1, default 0.2 (20% random failures)
For the real SMTP provider set SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, and SMTP_FROM.

Stopping
bashdocker compose down -v
The -v flag removes the volumes so the next up starts with a clean database.
