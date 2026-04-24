# Order & Payment gRPC Microservices

## Architecture

```
[Client] --REST--> [Order Service :8080] --gRPC--> [Payment Service :9091]
                   [Order gRPC   :9090]
```

## Quick Start

```bash
docker-compose up --build
```

## API Endpoints

### Order Service (HTTP :8080)
- `POST /orders` — Create order (triggers gRPC payment)
- `GET /orders/:id` — Get order by ID
- `PATCH /orders/:id/cancel` — Cancel order

### Payment Service (HTTP :8081)
- `GET /payments/:order_id` — Get payment by order ID

## gRPC Streaming

Subscribe to real-time order updates:

```bash
go run order-service/cmd/grpc-client/main.go --addr localhost:9090 --order <order_id>
```

## Proto Repositories

- Protos: https://github.com/Uzbekbay/order-grpc
- Generated: pb files are embedded locally in each service under `/pb` and `/paymentpb`

## Environment Variables

### Order Service
| Variable | Default | Description |
|---|---|---|
| DATABASE_URL | localhost orderdb | PostgreSQL connection |
| PAYMENT_GRPC_ADDRESS | localhost:9091 | Payment service gRPC address |
| ORDER_SERVICE_PORT | 8080 | HTTP port |
| ORDER_GRPC_PORT | 9090 | gRPC port |

### Payment Service
| Variable | Default | Description |
|---|---|---|
| DATABASE_URL | localhost paymentdb | PostgreSQL connection |
| PAYMENT_SERVICE_PORT | 8081 | HTTP port |
| PAYMENT_GRPC_PORT | 9091 | gRPC port |
