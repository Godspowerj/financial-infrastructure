# Module 01 — Payments Service & State Machine

## Engineering Problem Answered
How to record payment requests safely with idempotency (preventing duplicate charges during network failures) and transition payments through a state machine lifecycle (`pending` -> `processing` -> `successful` / `failed`) using HTTP integrations and webhooks.

## Architecture & System Design
The Payments service sits between clients and external bank/card processing networks.

```
Client (Postman / App) ────> Payments Service (:8080) ────> Bank Simulator (:8081)
                                      │                              │
                                      ▼                              ▼
                                 Postgres DB                 Async Webhook Callback
```

## Key Code Entry Points & File Map

1. **[payment.go](file:///c:/Users/USER/Downloads/fintech-lab-scaffold/services/payments/internal/payment.go)** — Domain Model & Status Constants
   - Defines struct `Payment` (amount stored in BIGINT kobo, status constants `StatusPending`, `StatusProcessing`, `StatusSuccessful`, `StatusFailed`).

2. **[gateway.go](file:///c:/Users/USER/Downloads/fintech-lab-scaffold/services/payments/internal/gateway.go)** — Bank Gateway Interface & Resilience
   - Interface `BankGateway` with `HTTPBankGateway` implementation.
   - Enforces 2s context timeouts, retries with exponential backoff (3 attempts), and catches `ErrBankUnreachable`.

3. **[repository.go](file:///c:/Users/USER/Downloads/fintech-lab-scaffold/services/payments/internal/repository.go)** — Postgres Persistence
   - `Create`: Inserts initial record as `pending`.
   - `GetByIdempotencyKey`: Prevents duplicate payment creation.
   - `UpdateStatus`: Updates payment status and `failure_reason` in PostgreSQL.

4. **[service.go](file:///c:/Users/USER/Downloads/fintech-lab-scaffold/services/payments/internal/service.go)** — Business Logic Core
   - Coordinates idempotency check, database insertion, and `ProcessPayment` state transitions.

5. **[handler.go](file:///c:/Users/USER/Downloads/fintech-lab-scaffold/services/payments/internal/handler.go)** — HTTP Router & Webhook Handler
   - `POST /payments`: Accepts client requests.
   - `POST /webhooks/bank`: Receives asynchronous settlement webhooks from the bank.

6. **[bank-simulator/cmd/api/main.go](file:///c:/Users/USER/Downloads/fintech-lab-scaffold/services/bank-simulator/cmd/api/main.go)** — Standalone Bank Server
   - Runs on port `:8081`, authorizes payments, and dispatches webhooks with retries.

## What Clicked Building This
- Idempotency ensures network drops don't double-charge users.
- Interfaces allow swapping `MockBankGateway` for `HTTPBankGateway` or `Paystack` without touching `service.go`.
- Microservices don't share databases; they communicate via HTTP APIs and Webhooks.
