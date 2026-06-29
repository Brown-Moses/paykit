# PayKit — Project Overview

**PayKit** is a production-ready, multi-tenant payment notification engine built in Go. It is designed to receive, verify, process, and forward Mobile Money (MTN MoMo) webhook events to registered merchants securely and reliably.

---

## What It Does

PayKit acts as a bridge between **MTN MoMo** (the payment provider) and **merchants** (businesses that accept MoMo payments).

### Core Workflow

1. **Receive** — MTN MoMo sends a payment webhook to `POST /webhook/momo/:merchant_id`.
2. **Verify** — PayKit validates the webhook's timestamp skew and `X-Signature` using HMAC-SHA256, checking for replay attacks using the unique `provider_tx_id`.
3. **Parse & Store** — The payload is parsed into a domain model, the payer's phone number (MSISDN) is SHA-256 hashed for privacy, and the transaction is persisted in PostgreSQL.
4. **Notify** — If the transaction is successful, PayKit asynchronously forwards a lightweight notification to the merchant's configured `webhook_url`.
5. **Retry** — Failed merchant deliveries are retried up to 3 times with exponential backoff.
6. **Dead Letter Queue (DLQ)** — If all 3 notifications fail, the notification is moved to the DLQ (`delivery_dlq` table) for operator inspection and manual retry.
7. **Log** — Every delivery attempt is recorded in a `delivery_logs` table for full observability.
8. **Query & Admin** — Merchants can query their transactions, scoped metrics, delivery logs, and manage their DLQ entries via a Bearer-token-protected REST API.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go 1.25 |
| **HTTP Framework** | [Gin](https://github.com/gin-gonic/gin) |
| **Database** | PostgreSQL 15 |
| **DB Driver** | [pgx/v5](https://github.com/jackc/pgx) (connection pooling via `pgxpool`) |
| **Instrumentation** | Prometheus Go Client |
| **API Documentation** | [Swagger (swaggo)](https://github.com/swaggo/swag) |
| **Environment Config** | [godotenv](https://github.com/joho/godotenv) |
| **Containerization** | Docker & Docker Compose |
| **Build Tool** | GNU Make |

---

## Project Structure

```
paykit/
├── cmd/paykit/main.go          # Application entry point with graceful shutdown
├── api/routes.go               # HTTP route definitions & middleware wiring
├── internal/
│   ├── auth/
│   │   ├── apikey.go           # API key generation (pk_live_...)
│   │   ├── middleware.go       # Bearer token authentication middleware
│   │   └── verifier.go         # HMAC-SHA256 signature, timestamp, & replay-attack verification
│   ├── metrics/
│   │   ├── metrics.go          # Prometheus custom metrics registration & tracking
│   │   └── http.go             # Prometheus scrape HTTP handler
│   ├── payments/
│   │   ├── notifier.go         # Async merchant notification with retry & DLQ logic
│   │   └── parser.go           # Webhook payload parsing & MSISDN hashing
│   ├── storage/
│   │   ├── models.go           # Domain structs (Transaction, Merchant, DeliveryLog, DLQ)
│   │   ├── store.go            # PostgreSQL queries & data access layer
│   │   └── migrate.sql         # Database schema, enums, indexes
│   └── webhook/
│       ├── handler.go          # HTTP handlers (webhooks, transactions, metrics, health)
│       ├── merchant.go         # Merchant registration handler
│       └── dlq_admin.go        # DLQ inspection and manual retry handlers
├── pkg/momodto/types.go        # MTN MoMo DTOs (WebhookPayload, NotifyPayload)
├── docs/                       # Auto-generated Swagger files
├── demo/postman/               # Postman collection for testing
├── docker-compose.yml          # Postgres 15 service definition
├── makefile                    # Docker, DB, app, and utility commands
└── go.mod / go.sum             # Go module definition
```

---

## Key Features

### Security
- **HMAC-SHA256 Signature Verification** — Every incoming MoMo webhook is cryptographically verified using a shared secret (`MOMO_WEBHOOK_SECRET`). Comparisons are constant-time to prevent timing attacks.
- **Replay Attack & Timestamp Protection** — Webhook requests are rejected if the timestamp is skew-stale (exceeding ±5 minutes) or if their `provider_tx_id` has already been processed.
- **IP Whitelisting** — Configurable incoming IP restrictions (`ALLOWED_IPS`) for MoMo webhook endpoints to ensure only MTN networks can trigger them.
- **Bearer Token Authentication** — Merchant-facing endpoints require a hashed `Bearer pk_live_...` API key.
- **Privacy** — Payer MSISDNs (phone numbers) are hashed with SHA-256 before storage.

### Reliability
- **Idempotency** — Duplicate webhooks are detected and silently acknowledged with `200 OK` so MTN does not retry unnecessarily.
- **Async Notifications** — Merchant webhook calls are performed in goroutines so the main request path never blocks.
- **Exponential Backoff Retries** — Failed merchant deliveries are retried 3 times (`1s`, `2s`, `4s` waits).
- **Dead Letter Queue (DLQ)** — Failed deliveries are quarantined in a `delivery_dlq` table after exhausting retries. Merchants can trigger manual retries on DLQ items which auto-resolve on success.
- **Delivery Observability** — Every attempt, response code, and error message is stored in `delivery_logs`.
- **Graceful Shutdown** — The HTTP server drains in-flight requests for up to 10 seconds on `SIGINT` / `SIGTERM`.

### Multi-Tenancy
- Merchants are first-class tenants. All transactions, metrics, and DLQ logs are scoped by `merchant_id`.
- Each merchant receives a unique, cryptographically random API key upon registration.

---

## API Endpoints

### Public (Unprotected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health & database connectivity check |
| `POST` | `/webhook/momo/:merchant_id` | Receive MTN MoMo payment webhook (IP Whitelisted) |
| `POST` | `/merchants` | Register a new merchant (returns API key) |
| `GET` | `/metrics/prometheus` | Prometheus metrics scrape endpoint |

### Protected (Bearer Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/transactions` | List/filter paginated transactions |
| `GET` | `/transactions/:id` | Get a single transaction by provider TX ID |
| `GET` | `/transactions/:id/deliveries` | Get webhook delivery attempts for a transaction |
| `GET` | `/metrics` | Get transaction & delivery metrics (scoped to merchant) |
| `GET` | `/admin/dlq` | List delivery DLQ entries for the merchant |
| `POST` | `/admin/dlq/:id/retry` | Trigger an immediate manual retry for a DLQ entry |

### Documentation

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/docs/index.html` | Swagger UI interactive documentation |

---

## Database Schema

### `transactions`
Stores every payment event received from MoMo.

| Column | Type | Notes |
|--------|------|-------|
| `id` | `BIGSERIAL PK` | Internal ID |
| `provider_tx_id` | `TEXT UNIQUE` | Idempotency key (MoMo's transaction ID) |
| `external_id` | `TEXT` | Merchant's order ID |
| `merchant_id` | `BIGINT FK → merchants` | Tenant scoping |
| `amount` | `NUMERIC(20,2)` | Transaction amount |
| `currency` | `VARCHAR(3)` | Default `RWF` |
| `status` | `tx_status ENUM` | `PENDING`, `SUCCESSFUL`, `FAILED` |
| `payer_msisdn` | `TEXT` | SHA-256 hashed phone number |
| `raw_payload` | `JSONB` | Full original webhook body (audit trail) |
| `received_at` | `TIMESTAMPTZ` | Event timestamp from MoMo |
| `created_at` | `TIMESTAMPTZ` | Record creation time |

### `merchants`
Stores tenant accounts.

| Column | Type | Notes |
|--------|------|-------|
| `id` | `BIGSERIAL PK` | Internal ID |
| `name` | `TEXT` | Business name |
| `api_key` | `TEXT UNIQUE` | Bearer token for API access |
| `webhook_url` | `TEXT` | URL to forward successful payments |
| `active` | `BOOLEAN` | Enable/disable flag |
| `created_at` | `TIMESTAMPTZ` | Registration time |

### `delivery_logs`
Stores every attempt to notify a merchant.

| Column | Type | Notes |
|--------|------|-------|
| `id` | `BIGSERIAL PK` | Internal ID |
| `transaction_id` | `BIGINT FK → transactions` | Related transaction |
| `merchant_id` | `BIGINT FK → merchants` | Related merchant |
| `webhook_url` | `TEXT` | Target URL |
| `attempt` | `INT` | Attempt number (0-indexed) |
| `status` | `delivery_status ENUM` | `SUCCESS`, `FAILED`, `RETRYING` |
| `response_code` | `INT` | HTTP status returned by merchant |
| `error_message` | `TEXT` | Network or HTTP error details |
| `delivered_at` | `TIMESTAMPTZ` | Attempt timestamp |

### `delivery_dlq`
Stores permanently failed merchant webhook notifications that require manual intervention.

| Column | Type | Notes |
|--------|------|-------|
| `id` | `BIGSERIAL PK` | Internal ID |
| `transaction_id` | `BIGINT FK → transactions` | Related transaction |
| `merchant_id` | `BIGINT FK → merchants` | Related merchant |
| `webhook_url` | `TEXT` | Target webhook URL |
| `attempt_count` | `INT` | Total delivery attempts tried |
| `last_error` | `TEXT` | Final attempt error message |
| `last_response_code`| `INT` | Final attempt HTTP response code |
| `status` | `dlq_status ENUM` | `PENDING`, `REQUEUED`, `RESOLVED`, `FAILED` |
| `available_at` | `TIMESTAMPTZ` | When item is eligible for retry |
| `created_at` | `TIMESTAMPTZ` | Quarantine timestamp |
| `resolved_at` | `TIMESTAMPTZ` | Resolve/manual delivery success timestamp |

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `MOMO_WEBHOOK_SECRET` | ✅ | — | Shared secret for HMAC verification |
| `PORT` | ❌ | `8080` | HTTP server port |

---

## Development Commands (Makefile)

```bash
# Infrastructure
make up          # Start Docker containers (Postgres)
make down        # Stop Docker containers
make ps          # Show container status

# Database
make migrate     # Run SQL migrations
make ping-db     # Check Postgres readiness
make db-backup   # Dump database to backups/
make db-restore FILE=backups/xxx.sql  # Restore from dump

# Application
make run         # Run the service locally
make build       # Compile all packages
make tidy        # Tidy Go modules
make swagger     # Regenerate Swagger docs

# Utilities
make health      # Curl the /health endpoint
```

---

## Data Flow Diagram (Simplified)

```
┌─────────────┐     POST /webhook/momo/:id     ┌─────────┐
│  MTN MoMo   │ ─────────────────────────────→ │  PayKit │
│  Provider   │      + X-Signature header      │  Server │
└─────────────┘                                └────┬────┘
                                                    │
                    ┌───────────────────────────────┼───────────────┐
                    │                               │               │
                    ▼                               ▼               ▼
            ┌───────────────┐              ┌─────────────┐   ┌─────────────┐
            │  HMAC Verify  │              │ Replay Check│   │ Parse JSON  │
            └───────┬───────┘              └──────┬──────┘   └──────┬──────┘
                    │                             │                 │
                    └───────────────┬─────────────┘                 │
                                    ▼                               ▼
                           ┌─────────────────┐            ┌─────────────────┐
                           │  Store in Postgres│          │  Hash MSISDN    │
                           │  (transactions)   │          │  (SHA-256)      │
                           └────────┬──────────┘          └─────────────────┘
                                    │
                                    ▼
                     ┌──────────────────────────┐
                     │  Async Notify Merchant     │
                     │  POST merchant.webhook_url │
                     └────────────┬───────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
               ┌────────┐   ┌────────┐   ┌────────┐
               │Attempt │   │Retry   │   │Retry   │
               │   1    │ → │   2    │ → │   3    │
               └────────┘   └────────┘   └────────┘
                    │
                    ▼
           ┌─────────────────┐
           │ Log to delivery │
           │     _logs       │
           └─────────────────┘
```

---

## Summary

PayKit is a secure, observable, and multi-tenant webhook ingestion layer for MTN MoMo. It solves the common problem of reliably accepting provider webhooks, verifying their authenticity, preventing duplicates, and forwarding clean payment events to downstream merchant systems — all while keeping sensitive payer data hashed and maintaining a full audit trail of every delivery attempt.

