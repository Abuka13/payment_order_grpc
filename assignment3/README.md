# Assignment 3 — Event-Driven Architecture with RabbitMQ

## Architecture Overview

```
[Client]
   |
   | HTTP POST /orders
   v
[Order Service :8080]
   |
   | gRPC ProcessPayment (order_id, amount, customer_email)
   v
[Payment Service :8081/:9091]
   |
   | Publishes to RabbitMQ (after DB commit)
   v
[RabbitMQ :5672] ── payment.completed queue (durable)
   |                       |
   |              (on rejection/failure)
   |                       v
   |              [DLQ: payment.completed.dead]
   |
   | Consumes (manual ACK, QoS=1)
   v
[Notification Service]
   |
   | Logs: [Notification] Sent email to user@example.com for Order #123. Amount: $99.99
```

## Services

| Service              | Port       | Role                          |
|----------------------|------------|-------------------------------|
| order-service        | 8080, 9092 | Creates orders, calls Payment via gRPC |
| payment-service      | 8081, 9091 | Processes payments, publishes events   |
| notification-service | —          | Consumes events, simulates email send  |
| postgres             | 5432       | Databases: orderdb, paymentdb, notificationdb |
| rabbitmq             | 5672, 15672| Message broker (Management UI at :15672) |

## How to Run

```bash
docker-compose up --build
```

Then create an order:
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "user-1",
    "item_name": "Laptop",
    "amount": 99900,
    "customer_email": "user@example.com"
  }'
```

Expected notification-service log:
```
[Notification] Sent email to user@example.com for Order #<uuid>. Amount: $999.00 Status: Authorized
```

## Idempotency Strategy

**Problem:** RabbitMQ guarantees *at-least-once* delivery — on consumer crash or network failure, a message can be redelivered. We must not send duplicate notifications.

**Solution:** PostgreSQL-backed `processed_events` table.

1. Each published `PaymentEvent` carries an `event_id` equal to `payment.TransactionID` (a UUID generated once per payment — globally unique, immutable).
2. Before processing, the consumer queries: `SELECT EXISTS(... WHERE event_id = $1)`.
3. If already seen → `msg.Ack()` without processing (silent deduplication).
4. After successful processing → `INSERT INTO processed_events (event_id)` with `ON CONFLICT DO NOTHING`.
5. Only after the DB insert succeeds → `msg.Ack(false)` is sent.

This order ensures: if the consumer crashes between processing and ACKing, the message will be redelivered, the `processed_events` check will catch it, and it will be silently ACKed.

## Manual ACK Logic

- `autoAck: false` is set when calling `ch.Consume(...)`.
- `ch.Qos(1, 0, false)` — consumer processes only one message at a time.
- **Success path:** process → mark in DB → `msg.Ack(false)`
- **Duplicate:** already in DB → `msg.Ack(false)` (remove from queue, no action)
- **Parse error / missing event_id:** `msg.Nack(false, false)` → message goes to DLQ
- **Transient DB error:** `msg.Nack(false, true)` → requeue for retry
- **Max retries exceeded:** `msg.Nack(false, false)` → DLQ

## Dead Letter Queue (Bonus)

- DLX (Dead Letter Exchange): `payment.completed.dlx` (fanout, durable)
- DLQ: `payment.completed.dead` (durable)
- Main queue is declared with `x-dead-letter-exchange: payment.completed.dlx`
- Messages rejected without requeue are routed to DLQ automatically
- Monitor DLQ via RabbitMQ Management UI: http://localhost:15672 (guest/guest)

## Reliability Guarantees

| Property            | Implementation |
|---------------------|----------------|
| Durable queues      | `durable: true` on all queue declarations |
| Persistent messages | `DeliveryMode: amqp.Persistent` in publisher |
| Publisher confirms  | `ch.Confirm(false)` + wait for broker ACK |
| At-least-once       | Manual ACK only after successful processing |
| Idempotency         | `processed_events` table with unique constraint |
| Graceful shutdown   | `os/signal` + `grpcServer.GracefulStop()` in all services |

## API

### Create Order (triggers full EDA flow)
```
POST /orders
{
  "customer_id": "string",
  "item_name": "string",
  "amount": 99900,          // in cents
  "customer_email": "string"
}
```

### Get Order
```
GET /orders/:id
```

### Create Payment directly
```
POST /payments
{
  "order_id": "string",
  "amount": 99900,
  "customer_email": "string"
}
```
