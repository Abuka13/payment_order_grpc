# Deployment & Quick Reference

## Quick Start (Development)

### 1. Database Setup

```bash
# Create databases
createdb orderdb
createdb paymentdb

# Run migrations
psql orderdb -f order-service/migrations/001_create_orders.sql
psql paymentdb -f payment-service/migrations/001_create_payments.sql
```

### 2. Install Dependencies

```bash
cd payment-service
go mod tidy

cd ../order-service
go mod tidy
```

### 3. Configure Environment (Linux/Mac)

```bash
# Payment Service
cd payment-service
export PAYMENT_SERVICE_PORT=8081
export PAYMENT_GRPC_PORT=9091
export DATABASE_URL="postgres://postgres:password@localhost:5432/paymentdb?sslmode=disable"

# Order Service (in another terminal)
cd order-service
export ORDER_SERVICE_PORT=8080
export ORDER_GRPC_PORT=9090
export PAYMENT_GRPC_ADDRESS=localhost:9091
export DATABASE_URL="postgres://postgres:password@localhost:5432/orderdb?sslmode=disable"
```

### 3. Configure Environment (Windows PowerShell)

```powershell
# Payment Service
cd payment-service
$env:PAYMENT_SERVICE_PORT="8081"
$env:PAYMENT_GRPC_PORT="9091"
$env:DATABASE_URL="postgres://postgres:password@localhost:5432/paymentdb?sslmode=disable"

# Order Service (in another terminal)
cd order-service
$env:ORDER_SERVICE_PORT="8080"
$env:ORDER_GRPC_PORT="9090"
$env:PAYMENT_GRPC_ADDRESS="localhost:9091"
$env:DATABASE_URL="postgres://postgres:password@localhost:5432/orderdb?sslmode=disable"
```

### 4. Start Services

```bash
# Terminal 1: Payment Service
cd payment-service
go run cmd/main.go
# Output: 
# Payment Service HTTP running on :8081
# Payment Service gRPC server running on :9091

# Terminal 2: Order Service
cd order-service
go run cmd/main.go
# Output:
# Order Service HTTP running on :8080
# Order Service gRPC server running on :9090
```

## Quick Test Commands

### Create Order
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"test","item_name":"item","amount":50000}'
```

### Get Order
```bash
curl http://localhost:8080/orders/ORDER_ID
```

### Subscribe to Order Updates (gRPC)
```bash
cd order-service/cmd/grpc-client
go run main.go -order-id=ORDER_ID
```

### Update Order Status (for testing)
```bash
psql orderdb -c "UPDATE orders SET status='Shipped' WHERE id='ORDER_ID';"
```

## File Structure

```
payment_order_grpc/
├── README_GRPC.md                    # Detailed gRPC documentation
├── CONTRACT_FIRST.md                 # Contract-First approach
├── TESTING_GUIDE.md                  # Complete test scenarios
├── ASSIGNMENT_2_SUMMARY.md           # Implementation summary
├── DEPLOYMENT.md                     # This file
│
├── order-service/
│   ├── cmd/
│   │   ├── main.go                   # Service entry point
│   │   └── grpc-client/
│   │       └── main.go               # gRPC streaming test client
│   ├── proto/
│   │   └── order.proto               # Proto definitions
│   ├── pb/
│   │   ├── order.pb.go              # Generated protobuf
│   │   └── order_grpc.pb.go         # Generated gRPC
│   ├── internal/
│   │   ├── config/config.go
│   │   ├── domain/order.go
│   │   ├── client/
│   │   │   ├── payment_http_client.go
│   │   │   └── payment_grpc_client.go
│   │   ├── repository/
│   │   ├── usecase/order_usecase.go
│   │   └── transport/
│   │       ├── http/order_handler.go
│   │       └── grpc/order_handler.go
│   ├── migrations/001_create_orders.sql
│   ├── go.mod
│   ├── go.sum
│   └── .env
│
└── payment-service/
    ├── cmd/
    │   └── main.go                   # Service entry point
    ├── proto/
    │   └── payment.proto             # Proto definitions
    ├── pb/
    │   ├── payment.pb.go            # Generated protobuf
    │   └── payment_grpc.pb.go       # Generated gRPC
    ├── internal/
    │   ├── config/config.go
    │   ├── domain/payment.go
    │   ├── repository/
    │   ├── usecase/payment_usecase.go
    │   └── transport/
    │       ├── http/payment_handler.go
    │       └── grpc/
    │           ├── payment_handler.go
    │           └── interceptor.go
    ├── migrations/001_create_payments.sql
    ├── go.mod
    ├── go.sum
    └── .env
```

## Environment Variables

### Payment Service

| Variable | Default | Purpose |
|----------|---------|---------|
| `PAYMENT_SERVICE_PORT` | 8081 | HTTP API port |
| `PAYMENT_GRPC_PORT` | 9091 | gRPC server port |
| `DATABASE_URL` | postgres://... | Database connection string |

### Order Service

| Variable | Default | Purpose |
|----------|---------|---------|
| `ORDER_SERVICE_PORT` | 8080 | HTTP API port |
| `ORDER_GRPC_PORT` | 9090 | gRPC server port |
| `PAYMENT_GRPC_ADDRESS` | localhost:9091 | Payment Service gRPC address |
| `DATABASE_URL` | postgres://... | Database connection string |

## Service Ports

| Service | HTTP | gRPC | Purpose |
|---------|------|------|---------|
| Order | 8080 | 9090 | External API + Streaming |
| Payment | 8081 | 9091 | Backward compat + Processing |

## Architecture

```
┌─────────────────────────────────────────┐
│          Clients (REST)                 │
└─────────────────┬───────────────────────┘
                  │
    ┌─────────────▼──────────────┐
    │   Order Service (HTTP)     │
    │   Port 8080                │
    └──────────┬────────────────┘
               │
    (gRPC) ┌───▼────────────────────┐
    :9090  │ Order Service (gRPC)   │
           │ - Streaming RPC        │
           └──────┬─────────────────┘
                  │
                  │ (gRPC)
                  │ :9091
    ┌─────────────▼──────────────┐
    │Payment Service (gRPC)      │
    │- ProcessPayment            │
    │- GetPaymentByOrderID       │
    └──────┬───────────────────┘
           │
    ┌──────▼──────────────────┐
    │Payment Service (HTTP)   │
    │Port 8081               │
    └────────────────────────┘
           │
    ┌──────▼──────────────────┐
    │  PostgreSQL Database   │
    │ - orderdb              │
    │ - paymentdb            │
    └────────────────────────┘
```

## Health Checks

### Check Payment Service Health

```bash
# HTTP health
curl -v http://localhost:8081/payments/test

# gRPC connectivity
grpcurl -plaintext localhost:9091 list
```

### Check Order Service Health

```bash
# HTTP health
curl -v http://localhost:8080/orders/test

# gRPC connectivity
grpcurl -plaintext localhost:9090 list
```

## Monitoring

### Payment Service Logs

Watch for:
```
[gRPC] Method: /payment.PaymentService/ProcessPayment, Duration: XXms
```

### Order Service Logs

Watch for:
```
[gRPC Streaming] Client subscribed to order XXX
[gRPC Streaming] Order XXX status changed from YYY to ZZZ
```

## Docker Deployment (Optional)

### Dockerfile for Payment Service

```dockerfile
FROM golang:1.22-alpine

WORKDIR /app
COPY payment-service . 

RUN go mod tidy
RUN go build -o payment-service cmd/main.go

EXPOSE 8081 9091

ENV PAYMENT_SERVICE_PORT=8081
ENV PAYMENT_GRPC_PORT=9091

CMD ["./payment-service"]
```

### Dockerfile for Order Service

```dockerfile
FROM golang:1.22-alpine

WORKDIR /app
COPY order-service .

RUN go mod tidy
RUN go build -o order-service cmd/main.go

EXPOSE 8080 9090

ENV ORDER_SERVICE_PORT=8080
ENV ORDER_GRPC_PORT=9090

CMD ["./order-service"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  payment-service:
    build: ./payment-service
    ports:
      - "8081:8081"
      - "9091:9091"
    environment:
      DATABASE_URL: "postgres://postgres:password@postgres:5432/paymentdb?sslmode=disable"
      PAYMENT_SERVICE_PORT: "8081"
      PAYMENT_GRPC_PORT: "9091"
    depends_on:
      - postgres

  order-service:
    build: ./order-service
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      DATABASE_URL: "postgres://postgres:password@postgres:5432/orderdb?sslmode=disable"
      ORDER_SERVICE_PORT: "8080"
      ORDER_GRPC_PORT: "9090"
      PAYMENT_GRPC_ADDRESS: "payment-service:9091"
    depends_on:
      - postgres
      - payment-service

volumes:
  postgres_data:
```

Run with:
```bash
docker-compose up -d
```

## Troubleshooting

### Services won't connect

```bash
# Check if ports are in use
lsof -i :8080  # Order HTTP
lsof -i :8081  # Payment HTTP
lsof -i :9090  # Order gRPC
lsof -i :9091  # Payment gRPC

# Kill process if needed
kill -9 <PID>
```

### Database connection fails

```bash
# Test connection
psql -U postgres -h localhost -d orderdb -c "SELECT 1"

# Check if databases exist
psql -l | grep -E "orderdb|paymentdb"

# Create if missing
createdb orderdb
createdb paymentdb

# Run migrations
psql orderdb -f order-service/migrations/001_create_orders.sql
psql paymentdb -f payment-service/migrations/001_create_payments.sql
```

### gRPC client can't connect

```bash
# Check if gRPC server is running
grpcurl -plaintext localhost:9091 list

# Should output:
# payment.PaymentService
# grpc.reflection.v1alpha.ServerReflection
```

## Performance Tuning

### Connection Pool Settings

Edit `internal/repository/postgres/order_postgres.go`:

```go
db.SetMaxOpenConns(25)      // Max concurrent connections
db.SetMaxIdleConns(5)       // Keep idle connections
db.SetConnMaxLifetime(5 * time.Minute)
```

### gRPC Buffer Sizes

Edit `cmd/main.go`:

```go
grpcServer := grpc.NewServer(
    grpc.MaxRecvMsgSize(4 * 1024 * 1024),  // 4MB
    grpc.MaxSendMsgSize(4 * 1024 * 1024),  // 4MB
)
```

## Backup & Recovery

### Backup Databases

```bash
pg_dump orderdb > orderdb_backup.sql
pg_dump paymentdb > paymentdb_backup.sql
```

### Restore Databases

```bash
createdb orderdb
psql orderdb < orderdb_backup.sql

createdb paymentdb
psql paymentdb < paymentdb_backup.sql
```

## References

- See `README_GRPC.md` for comprehensive documentation
- See `TESTING_GUIDE.md` for test scenarios
- See `CONTRACT_FIRST.md` for gRPC design patterns
- See `ASSIGNMENT_2_SUMMARY.md` for implementation details

