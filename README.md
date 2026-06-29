# PayKit

A production-ready, multi-tenant payment notification engine built in Go for MTN MoMo webhook processing.

## What It Does

PayKit acts as a bridge between **MTN MoMo** (the payment provider) and **merchants** (businesses that accept MoMo payments).

### Core Workflow

1. **Receive** — MTN MoMo sends a payment webhook to `POST /webhook/momo/:merchant_id`.
2. **Verify** — PayKit validates the webhook's timestamp and `X-Signature` using HMAC-SHA256, checking for replay attacks.
3. **Parse & Store** — The payload is parsed into a domain model, the payer's phone number (MSISDN) is SHA-256 hashed for privacy, and the transaction is persisted in PostgreSQL.
4. **Notify** — If the transaction is successful, PayKit asynchronously forwards a lightweight notification to the merchant's configured `webhook_url`.
5. **Retry** — Failed merchant deliveries are retried up to 3 times with exponential backoff.
6. **DLQ** — Webhooks that permanently fail delivery after retries are stored in the Dead Letter Queue (`delivery_dlq` table) for manual inspection and retries.
7. **Log & Instrument** — Every delivery attempt is recorded in a `delivery_logs` table, and system metrics are collected via Prometheus.
8. **Query & Admin** — Merchants can query their transactions, scoped metrics, and manage DLQ entries via a Bearer-token-protected REST API.

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go 1.25 |
| **HTTP Framework** | [Gin](https://github.com/gin-gonic/gin) |
| **Database** | PostgreSQL 15 |
| **DB Driver** | [pgx/v5](https://github.com/jackc/pgx) (connection pooling via `pgxpool`) |
| **API Documentation** | [Swagger (swaggo)](https://github.com/swaggo/swag) |
| **Environment Config** | [godotenv](https://github.com/joho/godotenv) |
| **Containerization** | Docker & Docker Compose |
| **Build Tool** | GNU Make |

## Project Structure

```
paykit/
├── cmd/paykit/main.go          # Application entry point with graceful shutdown
├── api/routes.go               # HTTP route definitions & middleware wiring
├── internal/
│   ├── auth/
│   │   ├── apikey.go           # API key generation (pk_live_...)
│   │   ├── middleware.go       # Bearer token authentication middleware
│   │   └── verifier.go         # HMAC-SHA256 signature & replay-attack verification
│   ├── payments/
│   │   ├── notifier.go         # Async merchant notification with retry logic
│   │   └── parser.go           # Webhook payload parsing & MSISDN hashing
│   ├── storage/
│   │   ├── models.go           # Domain structs (Transaction, Merchant, DeliveryLog)
│   │   ├── store.go            # PostgreSQL queries & data access layer
│   │   └── migrate.sql         # Database schema, enums, indexes
│   └── webhook/
│       ├── handler.go          # HTTP handlers (webhooks, transactions, health)
│       └── merchant.go         # Merchant registration handler
├── pkg/momodto/types.go        # MTN MoMo DTOs (WebhookPayload, NotifyPayload)
├── docs/                       # Auto-generated Swagger files
├── demo/postman/               # Postman collection for testing
├── docker-compose.yml          # Postgres 15 service definition
├── makefile                    # Docker, DB, app, and utility commands
└── go.mod / go.sum             # Go module definition
```

## Prerequisites

- Go 1.25 or higher
- PostgreSQL 15 or higher
- Docker & Docker Compose

## Installation

### 1. Clone the repository

```bash
git clone <repository-url>
cd paykit
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Set up environment

Create `.env` file with required variables:

```env
DATABASE_URL=postgres://paykit:paykit_secret@localhost:5434/paykit?sslmode=disable
MOMO_WEBHOOK_SECRET=your_secret_from_mtn_portal
PORT=8080
```

### 4. Start PostgreSQL

```bash
make up
```

### 5. Run database migrations

```bash
make migrate
```

## Usage

### Development Commands

```bash
# Infrastructure
make up          # Start Docker containers (Postgres)
make down        # Stop Docker containers
make ps          # Show container status

# Database
make migrate     # Run SQL migrations
make ping-db     # Check Postgres readiness

# Application
make run         # Run the service locally
make build       # Compile all packages
make tidy        # Tidy Go modules

# Utilities
make health      # Curl the /health endpoint
```

### API Endpoints

#### Public (Unprotected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health & database connectivity check |
| `POST` | `/webhook/momo/:merchant_id` | Receive MTN MoMo payment webhook (IP Whitelisted) |
| `POST` | `/merchants` | Register a new merchant (returns API key) |
| `GET` | `/metrics/prometheus` | Prometheus system metrics scrape endpoint |

#### Protected (Bearer Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/transactions` | List/filter paginated transactions |
| `GET` | `/transactions/:id` | Get a single transaction by provider TX ID |
| `GET` | `/transactions/:id/deliveries` | Get webhook delivery attempts for a transaction |
| `GET` | `/metrics` | Get transaction & delivery metrics (scoped to merchant) |
| `GET` | `/admin/dlq` | List delivery DLQ entries for the merchant |
| `POST` | `/admin/dlq/:id/retry` | Trigger manual retry for a DLQ entry |

#### Documentation

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/docs/index.html` | Swagger UI interactive documentation |

## Database

The service uses PostgreSQL for persistent storage. Migrations are applied via `make migrate`.

Key tables:
- `transactions` - Payment events from MTN MoMo
- `merchants` - Registered merchant accounts
- `delivery_logs` - Webhook delivery attempt records
- `delivery_dlq` - Permanently failed deliveries (Dead Letter Queue)

## Docker

Build and run with Docker Compose:

```bash
# Start all services
make up

# View logs
docker-compose logs -f paykit

# Stop services
make down
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `MOMO_WEBHOOK_SECRET` | ✅ | — | Shared secret for HMAC verification |
| `PORT` | ❌ | `8080` | HTTP server port |
| `ALLOWED_IPS` | ❌ | — | Comma-separated CIDRs for IP whitelisting |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions, please open an issue on the repository.
