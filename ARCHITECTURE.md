# Architecture & Data Flow Diagrams

## System Architecture (Assignment 2 - gRPC)

```
┌────────────────────────────────────────────────────────────────────┐
│                        External World                              │
├────────────────────────────────────────────────────────────────────┤
│  - Web Browser                                                     │
│  - Mobile App                                                      │
│  - CLI Client                                                      │
│  - Other Services                                                  │
└──────────────────────────┬─────────────────────────────────────────┘
                           │
                           │ HTTP/REST
                           │
        ┌──────────────────▼──────────────────┐
        │    Order Service                   │
        │    (Dual Server Architecture)      │
        ├──────────────────────────────────┐ │
        │  REST API (:8080)                │ │
        │  ├─ POST /orders                 │ │
        │  ├─ GET /orders/:id              │ │
        │  └─ PATCH /orders/:id/cancel     │ │
        └────────────┬─────────────────────┘ │
        │            │                       │
        │  ┌─────────▼─────────┐             │
        │  │  gRPC Server      │             │
        │  │  (:9090)          │             │
        │  │  - Streaming RPC  │             │
        │  │  - SubscribeToOrder│            │
        │  │    Updates        │             │
        │  └──────────────────┘             │
        │                                   │
        │  Internal Structure:             │
        │  ├─ Config Layer                │
        │  ├─ Domain Models               │
        │  ├─ Use Cases                   │
        │  ├─ Repositories                │
        │  ├─ HTTP Handlers               │
        │  └─ gRPC Handlers               │
        └─────────────┬────────────────────┘
                      │
                      │ gRPC/ProtoBuf
                      │ (:9091)
        ┌─────────────▼────────────────────┐
        │   Payment Service               │
        │   (Dual Server Architecture)    │
        ├──────────────────────────────┐  │
        │  gRPC Server (:9091)         │  │
        │  ├─ ProcessPayment RPC       │  │
        │  ├─ GetPaymentByOrderID RPC  │  │
        │  └─ [LoggingInterceptor]     │  │
        └────────────┬──────────────────┘ │
        │            │                    │
        │  ┌─────────▼────────┐           │
        │  │  REST API        │           │
        │  │  (:8081)         │           │
        │  │  - Backward compat│           │
        │  └──────────────────┘           │
        │                                 │
        │  Internal Structure:           │
        │  ├─ Config Layer              │
        │  ├─ Domain Models             │
        │  ├─ Use Cases                 │
        │  ├─ Repositories              │
        │  ├─ HTTP Handlers             │
        │  └─ gRPC Handlers             │
        └──────────────┬─────────────────┘
                       │
                       │ SQL
                       │
        ┌──────────────▼──────────────────┐
        │   PostgreSQL Databases          │
        │  ┌──────────┐  ┌──────────┐    │
        │  │ orderdb  │  │paymentdb │    │
        │  ├──────────┤  ├──────────┤    │
        │  │ orders   │  │ payments │    │
        │  │ table    │  │ table    │    │
        │  └──────────┘  └──────────┘    │
        └─────────────────────────────────┘
```

## Service Communication Flow

### Flow 1: REST Order Creation → gRPC Payment Processing

```
┌─────────┐
│ Client  │
└────┬────┘
     │
     │ HTTP POST /orders
     │ {customer_id, item_name, amount}
     │
     ▼
┌────────────────────────────────────────┐
│  Order Service (HTTP)                  │
│  ├─ Validate input                     │
│  ├─ Create order in DB (status=Pending)│
│  └─ Call payment service               │
└────────────┬───────────────────────────┘
             │
             │ gRPC Call (ProtoBuf)
             │ ProcessPayment(order_id, amount)
             │
             ▼
       ┌────────────────────────────────┐
       │ Payment Service (gRPC)         │
       │ ├─ PaymentUsecase.Create()     │
       │ ├─ Validate amount             │
       │ ├─ Determine status            │
       │ │  (>100k → Declined)          │
       │ │  (≤100k → Authorized)        │
       │ ├─ Store in DB                 │
       │ └─ Return PaymentResponse      │
       └────────────┬───────────────────┘
                    │
                    │ gRPC Response
                    │ PaymentResponse
                    │ {status, transaction_id}
                    │
             ┌──────▼────────────────────┐
             │ Order Service             │
             │ ├─ Check payment status   │
             │ ├─ If Authorized:        │
             │ │  └─ Update order=Paid   │
             │ ├─ If Declined:          │
             │ │  └─ Update order=Failed │
             │ └─ Return Order to client │
             └──────┬───────────────────┘
                    │
                    │ HTTP Response
                    │ Order {status, ...}
                    │
             ┌──────▼──────┐
             │   Client    │
             └─────────────┘
```

### Flow 2: gRPC Streaming - Real-time Order Updates

```
┌──────────────────────────────────┐
│  gRPC Client                     │
│  (e.g., grpcurl, test app)       │
└────────────┬─────────────────────┘
             │
             │ gRPC Connection
             │ SubscribeToOrderUpdates(order_id)
             │
             ▼
┌──────────────────────────────────────┐
│  Order Service (gRPC Server)         │
│  ├─ Verify order exists              │
│ │ ├─ Send initial status             │
│ │ └─ Enter polling loop:             │
│ │     ├─ Query DB every 2 seconds    │
│ │     ├─ Check status change         │
│ │     │  YES → Send update            │
│ │     │  NO  → Continue polling      │
│ │     └─ Timeout after 5 min         │
└────────────┬───────────────────────┘
             │
             │ gRPC Stream
             │ OrderStatusUpdate
             │ (repeated)
             │
             ▼
    ┌────────────────────┐
    │ gRPC Client        │
    │ ├─ Receive update  │
    │ └─ Display status  │
    └────────────────────┘
```

Example stream updates:
```
Time 0s:   status=Paid     message="Initial status"
Time 5s:   status=Processing    message="Status changed to Processing"
Time 12s:  status=Shipped  message="Status changed to Shipped"
Time 300s: [Connection timeout - client disconnects]
```

## Data Flow: Order Processing

```
┌─────────────────────────────────────────────────────────┐
│            Database Tables                              │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  orderdb/orders:                                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │ id  │ customer_id │ amount │ status  │ created_at │   │
│  ├─────┼─────────────┼────────┼─────────┼────────────┤   │
│  │ A   │ cust_001    │ 50000  │ Pending │ 10:30:00   │   │
│  └─────┴─────────────┴────────┴─────────┴────────────┘   │
│                        │                                  │
│                        │ gRPC ProcessPayment              │
│                        │                                  │
│  paymentdb/payments:   ▼                                  │
│  ┌────────────────────────────────────────────────┐      │
│  │ id  │ order_id │ transaction_id │ status      │      │
│  ├─────┼──────────┼────────────────┼─────────────┤      │
│  │ P1  │ A        │ trans_xxx      │ Authorized  │      │
│  └─────┴──────────┴────────────────┴─────────────┘      │
│                        │                                 │
│                        │ Update order status             │
│                        │                                 │
│  orderdb/orders:       ▼                                 │
│  ┌─────────────────────────────────────────────────┐    │
│  │ id  │ customer_id │ amount │ status  │ created_at │   │
│  ├─────┼─────────────┼────────┼─────────┼────────────┤   │
│  │ A   │ cust_001    │ 50000  │ Paid    │ 10:30:00   │   │
│  └─────┴─────────────┴────────┴─────────┴────────────┘   │
│                        │                                 │
│                        │ Database polling               │
│                        │ (every 2 seconds)              │
│                        ▼                                 │
│                   [gRPC Stream]                          │
│                   OrderStatusUpdate                      │
│                   status="Paid"                          │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## gRPC Service Definitions

### Payment Service (payment.proto)

```
┌──────────────────────────────────────────────┐
│  service PaymentService                      │
├──────────────────────────────────────────────┤
│                                              │
│  rpc ProcessPayment                          │
│    request:  PaymentRequest                 │
│    ├─ order_id: string                       │
│    └─ amount: int64                          │
│                                              │
│    response: PaymentResponse                │
│    ├─ id: string                             │
│    ├─ order_id: string                       │
│    ├─ transaction_id: string                 │
│    ├─ amount: int64                          │
│    ├─ status: string                         │
│    └─ created_at: Timestamp                 │
│                                              │
│  rpc GetPaymentByOrderID                    │
│    request:  GetPaymentRequest              │
│    └─ order_id: string                       │
│                                              │
│    response: PaymentResponse                │
│    (same as ProcessPayment)                 │
│                                              │
└──────────────────────────────────────────────┘
```

### Order Service (order.proto)

```
┌──────────────────────────────────────────────┐
│  service OrderService                        │
├──────────────────────────────────────────────┤
│                                              │
│  rpc SubscribeToOrderUpdates                │
│    request:  OrderRequest                   │
│    └─ order_id: string                       │
│                                              │
│    response: stream OrderStatusUpdate       │
│    ├─ order_id: string                       │
│    ├─ status: string                         │
│    ├─ updated_at: Timestamp                 │
│    └─ message: string                        │
│                                              │
│    (Streaming - multiple responses)         │
│                                              │
└──────────────────────────────────────────────┘
```

## Interceptor Flow

### Payment Service Request Logging

```
Request arrives at gRPC server
        │
        ▼
[LoggingInterceptor.UnaryInterceptor]
        │
        ├─ Record start time
        ├─ Log method name: /payment.PaymentService/ProcessPayment
        ├─ Log request data
        │
        ▼
[Route to actual handler]
        │
        ├─ ProcessPayment handler executes
        │
        ▼
[Back to interceptor]
        │
        ├─ Calculate duration
        ├─ Log duration in milliseconds
        ├─ Log error (if any)
        │
        ▼
Response sent to client
        
Example log:
[gRPC] Method: /payment.PaymentService/ProcessPayment, Request: ...
[gRPC] Method: /payment.PaymentService/ProcessPayment, Duration: 42 ms, Error: <nil>
```

## Clean Architecture Layers (Preserved from Assignment 1)

```
┌─────────────────────────────────────────┐
│  Transport Layer                        │
│  ├─ HTTP Handlers (REST)                │
│  └─ gRPC Handlers (RPC/Streaming)       │
├─────────────────────────────────────────┤
│  Use Case Layer                         │
│  ├─ OrderUsecase                        │
│  └─ PaymentUsecase                      │
│  (Business logic - UNCHANGED)           │
├─────────────────────────────────────────┤
│  Domain Layer                           │
│  ├─ Order entity                        │
│  └─ Payment entity                      │
│  (Validation rules - UNCHANGED)         │
├─────────────────────────────────────────┤
│  Repository Layer                       │
│  ├─ OrderRepository                     │
│  ├─ PaymentRepository                   │
│  └─ PostgreSQL implementation           │
│  (Data persistence - UNCHANGED)         │
└─────────────────────────────────────────┘
```

## Configuration & Dependency Injection

```
┌──────────────────────────────────────────────┐
│  main.go                                     │
│  ├─ Load .env variables                      │
│  ├─ Connect to database                      │
│  ├─ Initialize repositories                  │
│  ├─ Initialize use cases                     │
│  ├─ Initialize gRPC/HTTP handlers            │
│  ├─ Start gRPC server (goroutine)            │
│  └─ Start HTTP server (main thread)          │
├──────────────────────────────────────────────┤
│                                              │
│  Environment Variables:                      │
│  ├─ SERVICE_PORT (HTTP)                     │
│  ├─ GRPC_PORT (gRPC)                        │
│  ├─ DATABASE_URL (PostgreSQL)               │
│  └─ PAYMENT_GRPC_ADDRESS (Order service)    │
│                                              │
└──────────────────────────────────────────────┘
```

## Error Handling Strategy

```
┌─────────────────────────────────────────┐
│  HTTP Error                             │
├─────────────────────────────────────────┤
│  Input validation → 400 Bad Request     │
│  Not found → 404 Not Found              │
│  Server error → 503 Service Unavailable │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  gRPC Error                             │
├─────────────────────────────────────────┤
│  Input validation → InvalidArgument     │
│  Not found → NotFound                   │
│  Server error → Internal                │
│  Connection issue → Unavailable         │
└─────────────────────────────────────────┘

Both use google.golang.org/grpc/status
```

## Deployment Topology

```
Development:
┌────────────────────────────────────────────┐
│         Localhost                          │
│  ┌──────────────┐   ┌──────────────┐      │
│  │ Order Svc    │   │ Payment Svc  │      │
│  │ :8080, :9090 │   │ :8081, :9091 │      │
│  └──────────────┘   └──────────────┘      │
│         │                  │               │
│         └──────────────────┴───────────┐   │
│                                        │   │
│                    ┌───────────────────▼─┐ │
│                    │ PostgreSQL Docker   │ │
│                    │ :5432              │ │
│                    └───────────────────┘ │
└────────────────────────────────────────────┘

Production (Docker Compose):
┌────────────────────────────────────────────┐
│         Docker Network                     │
│  ┌──────────────┐   ┌──────────────┐      │
│  │ Order Svc    │   │ Payment Svc  │      │
│  │ Container    │   │ Container    │      │
│  └──────────────┘   └──────────────┘      │
│         │                  │               │
│         └──────────────────┴───────────┐   │
│                                        │   │
│                    ┌───────────────────▼─┐ │
│                    │ PostgreSQL Container│ │
│                    │ :5432              │ │
│                    └───────────────────┘ │
└────────────────────────────────────────────┘
```

This ASCII architecture documents all key aspects of the Assignment 2 implementation.

