# AP2 Assignment 1 – Clean Architecture based Microservices

## Overview
This project contains two microservices:
- Order Service
- Payment Service

The system follows Clean Architecture and REST-based synchronous communication.

## Architecture
Each service contains:
- domain
- usecase
- repository
- transport/http
- cmd (composition root)

Business logic is placed in the use case layer.
Handlers are thin and only parse requests and return responses.
Repositories are responsible for persistence.

## Bounded Contexts
### Order Service
Responsible for:
- creating orders
- retrieving orders
- cancelling pending orders

### Payment Service
Responsible for:
- authorizing or declining payments
- storing payment records
- returning payment status

## Communication
Order Service calls Payment Service via REST using a custom http.Client with a timeout of 2 seconds.

## Failure Handling
If Payment Service is unavailable:
- Order Service does not hang indefinitely
- timeout is triggered
- Order Service returns 503 Service Unavailable
- order is marked as Failed

## Databases
Each service has its own PostgreSQL database:
- orderdb
- paymentdb

No shared database and no shared models are used.

## Business Rules
- amount is int64
- amount must be greater than 0
- payments above 100000 are declined
- paid orders cannot be cancelled
- only pending orders can be cancelled

## Endpoints

### Order Service
- POST /orders
- GET /orders/:id
- PATCH /orders/:id/cancel

### Payment Service
- POST /payments
- GET /payments/:order_id