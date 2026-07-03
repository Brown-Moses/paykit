# PayKit

> **Production-ready MTN MoMo webhook engine for East African startups.**  
> Built in Go. Self-hosted. Ships with a full developer portal out of the box.

PayKit sits between MTN MoMo and your business — receiving payment webhooks, verifying them cryptographically, storing them, and forwarding clean notifications to your app. It handles the hard parts: duplicate prevention, replay attacks, failed delivery retries, and a dead letter queue — so your team doesn't have to.

---

## Portals

### Merchant Dashboard (`/portal/dashboard.html`)
![Merchant Dashboard](docs/screenshots/dashboard.png)

Every merchant gets a self-service portal with:
- Monthly webhook usage meter (calls used vs. plan limit)
- Webhook success rate and volume trend chart
- Self-managed webhook URL configuration
- Transaction history and delivery logs
- DLQ retry panel for failed deliveries
- Subscription tier overview

### Operator Portal (`/operator/`)
![Operator Portal](docs/screenshots/operator.png)

A private control panel for you as the platform operator:
- All merchants overview with live usage stats
- Activate / deactivate merchant accounts
- Quota management per merchant
- Platform-wide Prometheus metrics dashboard

---

## How It Works

```
MTN MoMo                PayKit                    Your App
    │                      │                          │
    │  POST /webhook/momo  │                          │
    │─────────────────────>│                          │
    │                      │ 1. Verify timestamp      │
    │                      │ 2. Verify HMAC-SHA256    │
    │                      │ 3. Check replay (tx_id)  │
    │                      │ 4. Hash MSISDN (privacy) │
    │                      │ 5. Save to PostgreSQL    │
    │                      │                          │
    │                      │  POST merchant.webhook   │
    │                      │─────────────────────────>│
    │                      │   async, with retries    │
    │  200 OK              │                          │
    │<─────────────────────│                          │
```

If your app is unreachable, PayKit retries 3 times with exponential backoff. If all attempts fail, the delivery is quarantined in the Dead Letter Queue and you can retry it manually from the portal.

---

## Core Features

### Security
- **HMAC-SHA256 signature verification** — every webhook cryptographically verified against a shared secret
- **Timestamp window validation** — rejects requests outside ±5 minutes (configurable)
- **Replay attack protection** — duplicate `transactionId` values silently acknowledged and ignored
- **Constant-time comparison** — prevents timing attacks on signature checks
- **IP whitelisting** — restrict webhook ingestion to MTN's IP ranges
- **MSISDN hashing** — payer phone numbers stored as SHA-256 hashes, never plaintext

### Reliability
- **Async notifications** — webhook processing never blocks on merchant delivery
- **Exponential backoff retries** — 3 attempts at 1s, 2s, 4s intervals
- **Dead Letter Queue** — permanently failed deliveries quarantined and retryable from the portal
- **Full delivery audit log** — every attempt, response code, and error stored in `delivery_logs`
- **Graceful shutdown** — drains in-flight requests for 10 seconds on `SIGTERM`

### Observability
- **Prometheus metrics** at `/metrics/prometheus` (Prometheus exposition format)
- **JSON metrics** at `/metrics` scoped per merchant
- **Swagger UI** at `/docs/index.html`
- **Health endpoint** at `/health`

### Multi-tenancy
- Every merchant is an isolated tenant — transactions, metrics, DLQ all scoped by `merchant_id`
- Unique `pk_live_...` API keys per merchant, stored as SHA-256 hashes
- Bearer token authentication on all merchant-facing endpoints

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.25 |
| HTTP Framework | Gin |
| Database | PostgreSQL 15 |
| DB Driver | pgx/v5 with pgxpool |
| Metrics | Prometheus Go Client |
| API Docs | Swagger (swaggo) |
| Config | godotenv |
| Containers | Docker & Docker Compose |

---

## Quick Start

### Prerequisites
- Go 1.25+
- Docker & Docker Compose

### 1. Clone

```bash
git clone https://github.com/Brown-Moses/paykit.git
cd paykit
```

### 2. Configure

```bash
cp .env.example .env
```

Edit `.env` — set these two required values:

```env
DATABASE_URL=postgres://paykit:paykit_secret@localhost:5434/paykit?sslmode=disable
MOMO_WEBHOOK_SECRET=your_secret_from_mtn_portal
ADMIN_PASSWORD=your_strong_operator_password
```

> **Warning:** Define `MOMO_WEBHOOK_SECRET` exactly once. Duplicate entries in `.env` cause silent signature failures.

### 3. Start

```bash
make up        # Start PostgreSQL
make migrate   # Run schema migrations
make run       # Start PayKit
```

### 4. Verify

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

### 5. Open the portals

| Portal | URL | Auth |
|--------|-----|------|
| Merchant Portal | `http://localhost:8080/portal/index.html` | Your `pk_live_...` API key |
| Operator Portal | `http://localhost:8080/operator/` | `ADMIN_PASSWORD` from `.env` |
| API Docs | `http://localhost:8080/docs/index.html` | None |

---

## Registering a Merchant

```bash
curl -X POST http://localhost:8080/merchants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Deliveries",
    "webhook_url": "https://your-app.com/webhooks/paykit"
  }'
```

```json
{
  "id": 2,
  "name": "Acme Deliveries",
  "api_key": "pk_live_abc123...",
  "webhook_url": "https://your-app.com/webhooks/paykit",
  "message": "store this api_key safely — it will not be shown again"
}
```

Point MTN MoMo at:
```
https://your-paykit.com/webhook/momo/2
```

---

## API Reference

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health and DB connectivity check |
| `POST` | `/merchants` | Register a new merchant |
| `POST` | `/webhook/momo/:merchant_id` | MTN MoMo webhook ingestion (IP whitelisted) |
| `GET` | `/metrics/prometheus` | Prometheus scrape endpoint |

### Protected Endpoints (Bearer Auth)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/transactions` | List transactions (paginated, filterable) |
| `GET` | `/transactions/:id` | Get transaction by provider TX ID |
| `GET` | `/transactions/:id/deliveries` | Get delivery attempts for a transaction |
| `GET` | `/metrics` | Merchant-scoped transaction metrics |
| `GET` | `/admin/dlq` | List DLQ entries for authenticated merchant |
| `POST` | `/admin/dlq/:id/retry` | Trigger manual retry for a DLQ entry |
| `POST` | `/auth/login` | Validate API key and return merchant profile |

### Portal Routes

| Path | Description |
|------|-------------|
| `/portal/*` | Merchant self-service portal (static HTML/JS/CSS) |
| `/operator/*` | Operator control panel (Basic Auth protected) |

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `MOMO_WEBHOOK_SECRET` | ✅ | — | Shared secret for HMAC-SHA256 verification |
| `ADMIN_PASSWORD` | ✅ | `admin` | Operator portal password |
| `PORT` | ❌ | `8080` | HTTP server port |
| `ALLOWED_IPS` | ❌ | — | Comma-separated CIDRs for webhook IP whitelist |
| `WEBHOOK_MAX_CLOCK_SKEW_SECONDS` | ❌ | `300` | Max timestamp age in seconds (±5 min default) |

---

## Database Schema

Four tables:

- **`merchants`** — tenant accounts, API keys, webhook URLs, plan limits
- **`transactions`** — payment events from MTN MoMo, one row per webhook
- **`delivery_logs`** — every delivery attempt, response code, and error message
- **`delivery_dlq`** — permanently failed deliveries awaiting manual retry

---

## Makefile Commands

```bash
make up          # Start Docker containers (PostgreSQL)
make down        # Stop containers
make migrate     # Apply database migrations
make run         # Run PayKit locally
make build       # Compile binary
make health      # Check /health endpoint
make db-backup   # Dump database to backups/
make swagger     # Regenerate Swagger docs
```

---

## Prometheus Metrics

After sending at least one webhook:

```bash
curl http://localhost:8080/metrics/prometheus | grep paykit
```

```
paykit_merchant_webhook_deliveries_total{merchant_id="2",status="SUCCESS"} 1
paykit_delivery_dlq_enqueues_total{merchant_id="2",reason="delivery_failed"} 0
paykit_delivery_dlq_retries_total{merchant_id="2",result="success"} 0
paykit_delivery_dlq_items_resolved_total{merchant_id="2"} 0
```

Connect Prometheus and Grafana by adding to `docker-compose.yml` — see `USAGE_GUIDE.md` for the full setup.

---

## Subscription Tiers

PayKit ships with four built-in tiers enforced at the engine level:

| Tier | Monthly Calls | Price |
|------|--------------|-------|
| Free | 1,000 | $0 |
| Starter | 3,000 | $15/mo |
| Growth | 15,000 | $50/mo |
| Enterprise | 100,000 | $150/mo |

Merchants hitting 80% of their limit receive a warning in the portal. At 100%, new webhook ingestion is paused until the next billing cycle or a tier upgrade.

---

## Notification Payload

What PayKit sends to your `webhook_url` on a successful payment:

```json
{
  "external_id": "ORDER-001",
  "status": "SUCCESSFUL",
  "amount": "5000",
  "currency": "RWF",
  "paid": true
}
```

---

## Project Structure

```
paykit/
├── cmd/paykit/main.go          # Entry point, graceful shutdown
├── api/routes.go               # Route definitions and middleware
├── internal/
│   ├── auth/                   # API key auth, HMAC verification, middleware
│   ├── metrics/                # Prometheus counters and HTTP handler
│   ├── payments/               # Async notifier, payload parser
│   ├── storage/                # PostgreSQL store, models, migrations
│   └── webhook/                # HTTP handlers, DLQ admin
├── web/
│   ├── portal/                 # Merchant portal (HTML/CSS/JS)
│   └── operator/               # Operator portal (HTML/CSS/JS)
├── pkg/momodto/                # MTN MoMo payload DTOs
├── docs/                       # Swagger generated files
├── demo/postman/               # Postman collection
├── docker-compose.yml
├── makefile
├── USAGE_GUIDE.md              # Full operator and merchant guide
└── go.mod
```

---

## Documentation

- **Full usage guide** → [`USAGE_GUIDE.md`](./USAGE_GUIDE.md)
- **API reference** → `http://localhost:8080/docs/index.html`
- **Live test procedures** → See `USAGE_GUIDE.md` → Live Production Testing

---

## License

MIT License — Copyright (c) 2026 Moses Wuruem Justin

See [`LICENSE`](./LICENSE) for full terms.

---

## Built By

**Moses Wuruem Justin** — Go backend engineer, Kigali, Rwanda.  
Building infrastructure tools for East African fintech.

> *PayKit is purpose-built for the MTN MoMo ecosystem in East Africa — where transaction sizes are small, margins matter, and every dropped webhook is a real business problem.*