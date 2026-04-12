# AP2 Assignment 2 – gRPC Migration & Contract-First Development

## Overview
This project demonstrates a migration from REST API to gRPC communication between microservices, following Clean Architecture principles and Contract-First development approach.

**Key Evolution from Assignment 1:**
- ✅ Contract-First approach using Protocol Buffers
- ✅ gRPC for inter-service communication
- ✅ Server-side streaming for real-time updates
- ✅ gRPC interceptors for monitoring and logging
- ✅ Environment-based configuration
- ✅ Maintains backward-compatible REST API

## Architecture

### High-Level Architecture
```
┌──────────────────────────────────────────────────────────┐
│              External Clients (REST)                     │
└─────────────────────┬──────────────────────────────────┘
                      │
        ┌─────────────▼──────────────┐
        │    Order Service           │
        │  ├─ REST API (:8080)      │
        │  ├─ gRPC Server (:9090)   │◄──────────┐
        │  │  └─ Streaming RPC      │           │ Real-time
        │  └─ Clean Architecture    │           │ Updates
        └──────────┬────────────────┘           │
                   │                            │
           (gRPC) ▼                             │
        ┌──────────────────────────┐            │
        │  Payment Service         │            │
        │  ├─ REST API (:8081)     │            │
        │  ├─ gRPC Server (:9091)  │────────────┘
        │  │  └─ Unary RPCs        │
        │  ├─ Interceptors         │
        │  └─ Clean Architecture   │
        └──────┬───────────────────┘
               │
        ┌──────▼─────────────┐
        │  PostgreSQL DB      │
        │ ├─ orderdb         │
        │ └─ paymentdb       │
        └────────────────────┘
```

### Service Architecture
Each service maintains Clean Architecture with:
- **Domain Layer:** Business entities (Order, Payment)
- **Use Case Layer:** Business logic unchanged from Assignment 1
- **Repository Layer:** Data persistence (PostgreSQL)
- **Transport Layer:** Both HTTP and gRPC handlers
- **Config Layer:** Environment-based configuration

## Key Features

### 1. gRPC Communication
- **Order Service → Payment Service:** gRPC unary calls for payment processing
- **Stronger typing:** Protocol Buffers enforce message contracts
- **Better performance:** Binary protocol vs JSON
- **Status codes:** Proper gRPC error handling

### 2. Server-Side Streaming
- **Real-time order updates:** SubscribeToOrderUpdates RPC
- **Database polling:** Checks for updates every 2 seconds
- **Automatic notifications:** Clients receive immediate status changes
- **No fake data:** Updates tied to actual database changes

### 3. gRPC Interceptors
- **Payment Service interceptor:** Logs all incoming requests
- **Monitoring:** Method name, duration, and errors
- **Performance tracking:** Response time in milliseconds

### 4. Environment Configuration
All services use environment variables (no hardcoded values):
- Ports: `ORDER_SERVICE_PORT`, `PAYMENT_GRPC_PORT`
- Addresses: `PAYMENT_GRPC_ADDRESS`
- Database: `DATABASE_URL`

## Bounded Contexts

### Order Service
**Responsibilities:**
- Creating orders with payment processing
- Retrieving orders by ID
- Cancelling pending orders
- Broadcasting real-time order status updates

**Communication:**
- REST API for external clients
- gRPC client for Payment Service calls
- gRPC server for order update streaming

### Payment Service
**Responsibilities:**
- Processing and authorizing/declining payments
- Storing payment records
- Retrieving payment status by order ID

**Communication:**
- REST API for backward compatibility
- gRPC server for order service calls
- Request logging via interceptor

## Business Rules
- Amount must be positive (> 0)
- Payments above 100,000 are declined
- Paid orders cannot be cancelled
- Only pending orders can be cancelled
- Order status updates trigger streaming notifications

## Endpoints

### Order Service (HTTP/REST)
- `POST /orders` - Create new order
- `GET /orders/:id` - Get order details
- `PATCH /orders/:id/cancel` - Cancel pending order

### Order Service (gRPC)
- `SubscribeToOrderUpdates(OrderRequest) -> stream OrderStatusUpdate` - Real-time updates

### Payment Service (HTTP/REST)
- `POST /payments` - Create payment record
- `GET /payments/:order_id` - Get payment by order ID

### Payment Service (gRPC)
- `ProcessPayment(PaymentRequest) -> PaymentResponse` - Unary RPC
- `GetPaymentByOrderID(GetPaymentRequest) -> PaymentResponse` - Unary RPC

## Proto Repositories

For Contract-First Development:
1. **Proto Repository:** Separate repo containing only `.proto` files
2. **Generated Code Repository:** Auto-generated code via GitHub Actions
3. **Services:** Import generated code from remote repositories

## Technology Stack
- **Language:** Go 1.22
- **gRPC:** v1.56.2
- **Protocol Buffers:** v3
- **Framework:** Gin (REST), gRPC (RPC)
- **Database:** PostgreSQL
- **Architecture:** Clean Architecture

## Migration from Assignment 1 to Assignment 2

### What Changed

| Aspect | Assignment 1 | Assignment 2 |
|--------|-------------|------------|
| **Inter-service Communication** | REST (HTTP) | gRPC |
| **Message Format** | JSON | Protocol Buffers |
| **Type Safety** | String-based | Strong typing |
| **Real-time Updates** | Client polling | Server-side streaming |
| **Request Logging** | Manual in handlers | Interceptors |
| **Configuration** | Hardcoded | Environment variables |
| **Order Status Subscription** | Not available | ✅ Available (gRPC) |

### Code Changes Summary

1. **Domain & Use Cases:** UNCHANGED
   - Business logic preserved entirely
   - Same validation rules
   - Same repositories

2. **Order Service:**
   - Added gRPC client for Payment Service
   - Added gRPC server with streaming
   - REST API maintained
   - Updated main.go for dual servers

3. **Payment Service:**
   - Added gRPC server
   - Added interceptor for logging
   - REST API maintained
   - Updated main.go for dual servers

4. **New Files:**
   - Proto definitions: `proto/*.proto`
   - Generated code: `pb/*.pb.go`
   - gRPC handlers: `transport/grpc/*`
   - Configuration: `internal/config/config.go`
   - gRPC client: `internal/client/payment_grpc_client.go`
   - Test client: `cmd/grpc-client/main.go`

## Getting Started

### Prerequisites
- Go 1.22+
- PostgreSQL 12+
- `protoc` compiler (optional - generated code is included)

### Quick Start

1. **Setup Databases:**
   ```bash
   createdb orderdb
   createdb paymentdb
   psql orderdb -f order-service/migrations/001_create_orders.sql
   psql paymentdb -f payment-service/migrations/001_create_payments.sql
   ```

2. **Install Dependencies:**
   ```bash
   cd payment-service && go mod tidy
   cd ../order-service && go mod tidy
   ```

3. **Set Environment Variables:**
   ```bash
   # Payment Service
   cd payment-service
   export PAYMENT_SERVICE_PORT=8081
   export PAYMENT_GRPC_PORT=9091
   export DATABASE_URL="postgres://postgres:password@localhost:5432/paymentdb?sslmode=disable"
   
   # Order Service
   cd order-service
   export ORDER_SERVICE_PORT=8080
   export ORDER_GRPC_PORT=9090
   export PAYMENT_GRPC_ADDRESS=localhost:9091
   export DATABASE_URL="postgres://postgres:password@localhost:5432/orderdb?sslmode=disable"
   ```

4. **Run Services:**
   ```bash
   # Terminal 1
   cd payment-service
   go run cmd/main.go
   
   # Terminal 2
   cd order-service
   go run cmd/main.go
   ```

## Further Reading

- See `README_GRPC.md` for detailed gRPC documentation
- See proto definitions: `order-service/proto/order.proto`
- See proto definitions: `payment-service/proto/payment.proto`
