# Payment & Order Service - gRPC Migration

## Overview

This is a microservice system that migrated from REST API communication to gRPC. The Order Service and Payment Service now communicate via gRPC, providing stronger typing and improved performance.

**Key Features:**
- ✅ REST API for external clients (Order Service)
- ✅ gRPC communication between services
- ✅ Server-side streaming for real-time order updates
- ✅ gRPC interceptors for logging and monitoring
- ✅ Environment-based configuration
- ✅ Clean Architecture with separated concerns

## Architecture

```
┌─────────────────────────────────────────────────────┐
│         External Client (REST)                      │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
         ┌─────────────────────────┐
         │   Order Service         │
         │  - REST API (:8080)     │
         │  - gRPC Server (:9090)  │◄────┐
         │                         │     │ Streaming
         └────────────┬────────────┘     │
                      │                  │
                      │ (gRPC)           │
                      │                  │
         ┌────────────▼────────────┐     │
         │  Payment Service        │     │
         │  - REST API (:8081)     │─────┘
         │  - gRPC Server (:9091)  │
         └─────────────────────────┘

         ┌─────────────────────────┐
         │    PostgreSQL DB         │
         │  (orders, payments)     │
         └─────────────────────────┘
```

## Technology Stack

- **Language:** Go 1.22
- **gRPC:** Protocol Buffers v3
- **Framework:** Gin (REST API)
- **Database:** PostgreSQL
- **Architecture:** Clean Architecture

## Project Structure

```
payment_order_grpc/
├── order-service/
│   ├── cmd/
│   │   ├── main.go                 # Order service entry point
│   │   └── grpc-client/           # gRPC streaming test client
│   ├── proto/
│   │   └── order.proto            # Proto definitions
│   ├── pb/
│   │   ├── order.pb.go           # Generated proto code
│   │   └── order_grpc.pb.go      # Generated gRPC code
│   ├── internal/
│   │   ├── config/               # Configuration helpers
│   │   ├── domain/               # Domain models
│   │   ├── client/               # gRPC payment client
│   │   ├── repository/           # Data layer
│   │   ├── usecase/              # Business logic
│   │   └── transport/
│   │       ├── http/             # REST handlers
│   │       └── grpc/             # gRPC handlers
│   ├── migrations/
│   ├── go.mod
│   └── .env
│
└── payment-service/
    ├── cmd/
    │   └── main.go               # Payment service entry point
    ├── proto/
    │   └── payment.proto         # Proto definitions
    ├── pb/
    │   ├── payment.pb.go        # Generated proto code
    │   └── payment_grpc.pb.go   # Generated gRPC code
    ├── internal/
    │   ├── config/              # Configuration helpers
    │   ├── domain/              # Domain models
    │   ├── repository/          # Data layer
    │   ├── usecase/             # Business logic
    │   └── transport/
    │       ├── http/            # REST handlers
    │       └── grpc/            # gRPC handlers (with interceptors)
    ├── migrations/
    ├── go.mod
    └── .env
```

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL 12+
- Protocol Buffers compiler (protoc)

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/Uzbekbay/payment_order_grpc.git
cd payment_order_grpc
```

2. **Setup databases:**
```bash
createdb orderdb
createdb paymentdb
```

3. **Run migrations:**
```bash
# Order Service migrations
psql orderdb -f order-service/migrations/001_create_orders.sql

# Payment Service migrations
psql paymentdb -f payment-service/migrations/001_create_payments.sql
```

4. **Configure environment variables:**

**Order Service (.env):**
```env
ORDER_SERVICE_PORT=8080
PAYMENT_GRPC_ADDRESS=localhost:9091
ORDER_GRPC_PORT=9090
DATABASE_URL=postgres://postgres:password@localhost:5432/orderdb?sslmode=disable
```

**Payment Service (.env):**
```env
PAYMENT_SERVICE_PORT=8081
PAYMENT_GRPC_PORT=9091
DATABASE_URL=postgres://postgres:password@localhost:5432/paymentdb?sslmode=disable
```

### Running Services

1. **Install dependencies:**
```bash
cd payment-service && go mod tidy
cd ../order-service && go mod tidy
```

2. **Start Payment Service:**
```bash
cd payment-service
go run cmd/main.go
```

Output:
```
2024/04/12 10:30:15 Payment Service HTTP running on :8081
2024/04/12 10:30:15 Payment Service gRPC server running on :9091
```

3. **Start Order Service (in another terminal):**
```bash
cd order-service
go run cmd/main.go
```

Output:
```
2024/04/12 10:30:20 Order Service HTTP running on :8080
2024/04/12 10:30:20 Order Service gRPC server running on :9090
```

## API Usage

### Create Order (REST)

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_123",
    "item_name": "Laptop",
    "amount": 50000
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust_123",
  "item_name": "Laptop",
  "amount": 50000,
  "status": "Paid",
  "created_at": "2024-04-12T10:30:20Z"
}
```

### Get Order (REST)

```bash
curl http://localhost:8080/orders/550e8400-e29b-41d4-a716-446655440000
```

### Cancel Order (REST)

```bash
curl -X PATCH http://localhost:8080/orders/550e8400-e29b-41d4-a716-446655440000/cancel
```

### Subscribe to Order Updates (gRPC Streaming)

Using the provided gRPC client:

```bash
cd order-service/cmd/grpc-client
go run main.go -order-id=550e8400-e29b-41d4-a716-446655440000 -addr=localhost:9090
```

Output:
```
Subscribed to order 550e8400-e29b-41d4-a716-446655440000. Waiting for updates...
[10:30:20] Order Status: Pending - Subscribed to order updates
[10:31:22] Order Status: Paid - Order status changed to Paid
```

### Using grpcurl (Alternative)

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Subscribe to order updates
grpcurl -plaintext \
  -d '{"order_id":"550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:9090 order.OrderService/SubscribeToOrderUpdates
```

## gRPC Services

### Payment Service

**Proto Definition:** `payment-service/proto/payment.proto`

**Methods:**
- `ProcessPayment(PaymentRequest) -> PaymentResponse` - Unary RPC
- `GetPaymentByOrderID(GetPaymentRequest) -> PaymentResponse` - Unary RPC

### Order Service

**Proto Definition:** `order-service/proto/order.proto`

**Methods:**
- `SubscribeToOrderUpdates(OrderRequest) -> stream OrderStatusUpdate` - Server-side streaming RPC

**Features:**
- Polls database every 2 seconds for updates
- Sends real-time updates to subscribers
- 5-minute timeout per subscription
- Proper error handling with gRPC status codes

## gRPC Interceptors

### Payment Service Interceptor

Logs all incoming gRPC requests with:
- Method name
- Request data
- Processing duration
- Error status

**Example log output:**
```
[gRPC] Method: /payment.PaymentService/ProcessPayment, Request: ...
[gRPC] Method: /payment.PaymentService/ProcessPayment, Duration: 45 ms, Error: <nil>
```

## Error Handling

### gRPC Status Codes Used

| Code | Scenario |
|------|----------|
| `InvalidArgument` | Invalid input parameters |
| `NotFound` | Resource not found |
| `Internal` | Database or processing errors |
| `Unavailable` | Service connection issues |

Example:
```go
return nil, status.Errorf(codes.InvalidArgument, "invalid order_id or amount")
```

## Configuration Management

Environment variables are loaded from `.env` files in each service directory.

**Priority:**
1. OS environment variables (highest)
2. `.env` file values
3. Default hardcoded values (lowest)

**Helper functions:**
- `config.GetEnv(key, defaultValue)` - Returns value or default
- `config.GetEnvOrFail(key)` - Returns value or exits

## Migration from REST to gRPC

### Key Changes:

1. **Order Service:**
   - REST API maintained for external clients
   - Added gRPC server for order streaming
   - Payment service calls now use gRPC client

2. **Payment Service:**
   - REST API maintained for backward compatibility
   - Added gRPC server for order processing
   - Added interceptor for request logging

3. **Data Flow:**
   - Client → REST API → Order Service ✓
   - Order Service → gRPC → Payment Service ✓
   - Order Service → gRPC Stream → Client ✓

### Benefits:

- **Stronger typing:** Protocol Buffers enforce message structure
- **Better performance:** Binary protocol vs JSON
- **Server-side streaming:** Real-time updates without polling
- **Automatic code generation:** Reduces manual RPC boilerplate
- **Backward compatibility:** REST API still available

## Database Schema

### Orders Table
```sql
CREATE TABLE orders (
    id VARCHAR(36) PRIMARY KEY,
    customer_id VARCHAR(100),
    item_name VARCHAR(255),
    amount BIGINT,
    status VARCHAR(50),
    created_at TIMESTAMP
);
```

### Payments Table
```sql
CREATE TABLE payments (
    id VARCHAR(36) PRIMARY KEY,
    order_id VARCHAR(36),
    transaction_id VARCHAR(36),
    amount BIGINT,
    status VARCHAR(50)
);
```

## Testing

### Integration Test: Create Order → Pay → Stream Updates

1. Create an order:
```bash
ORDER_ID=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"test","item_name":"item","amount":10000}' | jq -r '.id')
```

2. In another terminal, subscribe to updates:
```bash
go run order-service/cmd/grpc-client/main.go -order-id=$ORDER_ID
```

3. Verify payment was processed:
```bash
curl http://localhost:8081/payments/$ORDER_ID
```

4. You should see the streaming client receive the status update.

## Monitoring & Logging

### gRPC Logs

All gRPC requests are logged with timing information:
```
[gRPC] Method: /payment.PaymentService/ProcessPayment, Duration: 45 ms
```

### Application Logs

- Service startup information
- Streaming connection events
- Error messages with full context

## Proto Repositories

For the assignment, Protocol Buffer files should be in separate repositories:

1. **Proto Repository:** `https://github.com/Uzbekbay/proto-definitions`
   - Contains only `.proto` files
   - Used for contract management

2. **Generated Code Repository:** `https://github.com/Uzbekbay/payment-grpc` and `https://github.com/Uzbekbay/order-grpc`
   - Contains auto-generated `.pb.go` files
   - Updated via GitHub Actions
   - Imported by services

## GitHub Actions Workflow (Remote Generation)

For automated code generation, create `.github/workflows/generate.yml`:

```yaml
name: Generate Proto Code
on:
  push:
    paths:
      - '**/*.proto'
jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
      - run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
      - run: protoc --go_out=. --go-grpc_out=. proto/*.proto
      - name: Push generated code
        run: |
          git config user.email "action@github.com"
          git add pb/
          git commit -m "Generated code from proto files"
          git push
```

## Troubleshooting

### Connection Refused

**Problem:** `connect: connection refused`

**Solution:** Ensure both services are running and ports are correct:
```bash
# Check payment service
lsof -i :9091

# Check order service  
lsof -i :9090
```

### Database Connection Error

**Problem:** `failed to connect to database`

**Solution:** Verify `.env` variables and database connection:
```bash
psql -U postgres -h localhost -d orderdb -c "SELECT 1"
```

### gRPC Streaming Timeout

**Problem:** Streaming client times out after 5 minutes

**Expected behavior:** Subscriptions time out after 5 minutes to prevent resource leaks.

**Workaround:** Reconnect to stream:
```bash
go run order-service/cmd/grpc-client/main.go -order-id=$ORDER_ID
```

## Performance Considerations

- **gRPC overhead:** ~2-5ms per call vs 20-50ms for REST
- **Streaming:** Efficient for real-time updates (no polling)
- **Database polling:** Every 2 seconds (configurable)
- **Timeouts:** 5 seconds for unary calls, 5 minutes for streams

## Future Enhancements

- [ ] Implement gRPC authentication (TLS/SSL)
- [ ] Add circuit breaker pattern
- [ ] Implement request/response compression
- [ ] Add service discovery (Consul/etcd)
- [ ] Implement request tracing (Jaeger)
- [ ] Add metrics collection (Prometheus)

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://protobuf.dev/)
- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [gRPC Status Codes](https://grpc.io/docs/guides/status-codes/)

## License

MIT

## Author

Uzbekbay Abilkaiyr  
Advanced Programming 2  
Assignment 2 - gRPC Migration & Contract-First Development  
Deadline: 12.04.2026

