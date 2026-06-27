# PayKit Usage Guide

## Overview

PayKit is a production-ready, multi-tenant payment notification engine that bridges MTN MoMo payments to your business systems. This guide covers everything you need to know to operate PayKit, integrate it with your applications, and run live production tests.

---

## Table of Contents

### For PayKit Operators
1. [Quick Start](#quick-start)
2. [Production Deployment](#production-deployment)
3. [Configuration](#configuration)
4. [Monitoring & Maintenance](#monitoring--maintenance)
5. [Troubleshooting](#troubleshooting)

### For Merchants (Customers)
1. [Merchant Registration](#merchant-registration)
2. [MTN MoMo Setup](#mtn-momo-setup)
3. [Webhook Integration](#webhook-integration)
4. [API Usage](#api-usage)
5. [Live Production Testing](#live-production-testing)
6. [Go-Live Checklist](#go-live-checklist)

---

## For PayKit Operators

### Quick Start

Get PayKit running locally in 5 minutes:

```bash
# 1. Clone and enter directory
git clone <your-repo>
cd paykit

# 2. Start PostgreSQL
make up

# 3. Run migrations
make migrate

# 4. Configure environment
cp .env.example .env
# Edit .env with your DATABASE_URL and MOMO_WEBHOOK_SECRET

# 5. Run the service
make run
```

Visit `http://localhost:8080/health` to verify it's working.

---

### Configuration

#### Required Environment Variables

```bash
# Database connection (required)
DATABASE_URL=postgres://paykit:paykit_secret@host:5432/paykit?sslmode=require

# MTN MoMo webhook secret (required)
# WARNING: Only define this ONCE in your .env file.
# godotenv loads the last value if duplicated — causes silent signature mismatches.
MOMO_WEBHOOK_SECRET=your_shared_secret_from_mtn_portal

# Server port (optional, defaults to 8080)
PORT=8080
```

#### Optional Security

```bash
# IP whitelisting for webhooks (recommended for production)
ALLOWED_IPS=196.47.12.0/24,196.47.13.0/24

# Timestamp validation window in seconds (default: 300 = ±5 minutes)
WEBHOOK_MAX_CLOCK_SKEW_SECONDS=300
```

> **Critical:** Never define `MOMO_WEBHOOK_SECRET` twice in `.env`. Duplicate entries cause silent signature failures during webhook validation.

---

### Production Deployment

#### Docker Deployment

```bash
# Build and deploy
docker-compose -f docker-compose.yml up -d

# Check logs
docker-compose logs -f paykit
```

#### Manual Deployment

```bash
# Build binary
make build

# Run with environment
DATABASE_URL="postgres://..." MOMO_WEBHOOK_SECRET="..." ./paykit
```

#### System Requirements

- **CPU**: 1 core minimum, 2+ recommended
- **RAM**: 512MB minimum, 1GB+ recommended
- **Storage**: 10GB+ for logs and database
- **Network**: Stable internet for webhook delivery

---

### Monitoring & Maintenance

#### Health Checks

```bash
curl http://your-paykit.com/health
```

#### Prometheus Metrics

PayKit exposes Prometheus-compatible metrics at `/metrics/prometheus`:

```bash
curl http://localhost:8080/metrics/prometheus | grep paykit
```

Key metrics exposed:

| Metric | Description |
|--------|-------------|
| `paykit_merchant_webhook_deliveries_total` | Delivery attempts by merchant and status |
| `paykit_delivery_dlq_enqueues_total` | Webhooks pushed to DLQ after max retries |
| `paykit_delivery_dlq_retries_total` | Manual DLQ retry attempts |
| `paykit_delivery_dlq_items_resolved_total` | Successfully resolved DLQ items |

#### Connecting Prometheus + Grafana

Add to your `docker-compose.yml`:

```yaml
prometheus:
  image: prom/prometheus:latest
  ports:
    - "9090:9090"
  volumes:
    - ./prometheus.yml:/etc/prometheus/prometheus.yml

grafana:
  image: grafana/grafana:latest
  ports:
    - "3000:3000"
```

Create `prometheus.yml` in your project root:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "paykit"
    static_configs:
      - targets: ["paykit:8080"]
    metrics_path: "/metrics/prometheus"
```

#### Database Maintenance

```bash
# Backup database
make db-backup

# Monitor table sizes
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### Log Analysis

```sql
SELECT
    status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM delivery_logs
WHERE delivered_at >= NOW() - INTERVAL '24 hours'
GROUP BY status;
```

---

### Troubleshooting

#### Common Issues

**`invalid signature` on webhook:**
- Check `MOMO_WEBHOOK_SECRET` is defined only once in `.env`
- Verify the secret used to sign matches the one the server loaded
- Ensure you are signing the raw body bytes, not a re-serialized version

**`invalid or stale timestamp` on webhook:**
- Timestamp must be within ±5 minutes of server time (configurable via `WEBHOOK_MAX_CLOCK_SKEW_SECONDS`)
- Always generate timestamp dynamically — never hardcode it in test scripts
- Timestamp must be RFC3339 format: `2026-06-27T10:00:00Z`

**`verification failed` (500) on webhook:**
- This is the default error for unhandled verifier errors
- Most common cause: `transactionId` field missing or empty in payload
- PayKit uses `transactionId` (not `financialTransactionId`) for replay detection
- Check your JSON field names match the DTO exactly

**Merchant notifications failing:**
- Verify merchant `webhook_url` is publicly accessible
- Check delivery_logs for specific error messages
- Use DLQ admin endpoints to retry failed deliveries

#### Debug Commands

```bash
# Check recent transactions
SELECT id, provider_tx_id, status, received_at
FROM transactions
ORDER BY received_at DESC
LIMIT 10;

# Check failed deliveries
SELECT t.provider_tx_id, dl.webhook_url, dl.error_message, dl.attempt
FROM delivery_logs dl
JOIN transactions t ON dl.transaction_id = t.id
WHERE dl.status = 'FAILED'
ORDER BY dl.delivered_at DESC
LIMIT 10;

# Check DLQ
SELECT id, transaction_id, merchant_id, attempt_count, last_error, status
FROM delivery_dlq
ORDER BY created_at DESC
LIMIT 10;
```

---

## For Merchants (Customers)

### Merchant Registration

```bash
curl -X POST http://your-paykit.com/merchants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Your Business Name",
    "webhook_url": "https://your-app.com/webhooks/paykit"
  }'
```

**Response:**
```json
{
  "id": 2,
  "name": "Your Business Name",
  "api_key": "pk_live_abc123def456",
  "webhook_url": "https://your-app.com/webhooks/paykit",
  "message": "store this api_key safely — it will not be shown again"
}
```

> **Important:** Save your `api_key` immediately. It is shown only once and cannot be retrieved.

---

### MTN MoMo Setup

1. Sign up at the MTN Developer Portal
2. Get your `webhook_secret` — this must match `MOMO_WEBHOOK_SECRET` in PayKit
3. Set your webhook URL in the MTN portal:
   ```
   https://your-paykit.com/webhook/momo/{your_merchant_id}
   ```
4. Test in MTN sandbox before going live

---

### Webhook Integration

PayKit sends a notification to your `webhook_url` after every successful payment.

#### Notification Payload

```json
{
  "event_type": "payment.successful",
  "merchant_id": 2,
  "transaction": {
    "provider_tx_id": "TX-123456789",
    "external_id": "ORDER-ABC-123",
    "amount": 5000.00,
    "currency": "RWF",
    "status": "SUCCESSFUL",
    "received_at": "2026-06-27T10:30:00Z"
  }
}
```

#### Example Handler (Node.js)

```javascript
app.post('/webhooks/paykit', (req, res) => {
  const { event_type, transaction } = req.body;
  if (event_type === 'payment.successful') {
    await updateOrder(transaction.external_id, 'paid');
  }
  res.status(200).send('OK');
});
```

#### Example Handler (Python)

```python
@app.route('/webhooks/paykit', methods=['POST'])
def paykit_webhook():
    data = request.get_json()
    if data['event_type'] == 'payment.successful':
        update_order(data['transaction']['external_id'], 'paid')
    return 'OK', 200
```

---

### API Usage

All protected endpoints require your API key:

```bash
Authorization: Bearer pk_live_abc123def456
```

#### Query Transactions

```bash
# List all transactions
curl -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/transactions

# Filter by status
curl -H "Authorization: Bearer pk_live_abc123def456" \
     "http://your-paykit.com/transactions?status=SUCCESSFUL"
```

#### Check Delivery Logs

```bash
curl -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/transactions/{id}/deliveries
```

#### DLQ Admin

```bash
# List failed deliveries
curl -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/admin/dlq

# Retry a specific failed delivery
curl -X POST -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/admin/dlq/{id}/retry
```

---

### Live Production Testing

Follow these steps in order. Each step must pass before moving to the next.

#### Prerequisites

- PayKit running locally (`make run`)
- Migrations applied (`make migrate`)
- A request inspection tool — use [RequestBin](https://requestbin.com) or [webhook.site](https://webhook.site)
- `.env` loaded correctly (verify no duplicate `MOMO_WEBHOOK_SECRET` entries)

---

#### Step 1 — Health Check

```bash
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}`

---

#### Step 2 — Create Test Merchant

Replace the `webhook_url` with your RequestBin URL:

```bash
curl -X POST http://localhost:8080/merchants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Shop",
    "webhook_url": "https://your-requestbin-url.oast.pro"
  }'
```

Save the `id` and `api_key` from the response.

---

#### Step 3 — Send a Valid Webhook

> **Critical notes before running:**
> - Use `transactionId` (not `financialTransactionId`) — PayKit's DTO field is `transactionId`
> - Generate timestamp dynamically — hardcoded timestamps will fail the 5-minute window check
> - Sign the exact raw body string — any difference between signed and sent body causes `invalid signature`

```bash
TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
TXID="TX-$(date +%s)"
BODY="{\"amount\":\"5000\",\"currency\":\"RWF\",\"externalId\":\"EXT-001\",\"transactionId\":\"$TXID\",\"status\":\"SUCCESSFUL\",\"timestamp\":\"$TS\"}"
SIG=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "your_webhook_secret" | awk '{print $2}')

curl -X POST http://localhost:8080/webhook/momo/{merchant_id} \
  -H "Content-Type: application/json" \
  -H "X-Signature: $SIG" \
  -d "$BODY"
```

Expected: `200 OK`  
Check RequestBin — the notification payload should arrive within seconds.

---

#### Step 4 — Test Replay Protection

Send the **exact same command** again immediately.

Expected: `200 OK` but RequestBin receives **nothing** — duplicate silently ignored.

This confirms `provider_tx_id` deduplication is working.

---

#### Step 5 — Test Timestamp Rejection

Send a webhook with a stale timestamp (older than 5 minutes):

```bash
BODY="{\"amount\":\"5000\",\"currency\":\"RWF\",\"externalId\":\"EXT-002\",\"transactionId\":\"TX-stale\",\"status\":\"SUCCESSFUL\",\"timestamp\":\"2020-01-01T00:00:00Z\"}"
SIG=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "your_webhook_secret" | awk '{print $2}')

curl -X POST http://localhost:8080/webhook/momo/{merchant_id} \
  -H "Content-Type: application/json" \
  -H "X-Signature: $SIG" \
  -d "$BODY"
```

Expected: `401 {"error":"invalid or stale timestamp"}`

---

#### Step 6 — Verify Prometheus Counters

```bash
curl http://localhost:8080/metrics/prometheus | grep paykit
```

Expected output after at least one successful webhook:

```
paykit_merchant_webhook_deliveries_total{merchant_id="2",status="SUCCESS"} 1
```

---

#### Step 7 — Force a DLQ Entry

Set your merchant `webhook_url` to an invalid/unreachable URL, then send a valid webhook. After 3 retry attempts (exponential backoff), check the DLQ:

```bash
curl -H "Authorization: Bearer your_api_key" \
     http://localhost:8080/admin/dlq
```

Expected: one record with `status: "FAILED"`

Then retry it:

```bash
curl -X POST \
     -H "Authorization: Bearer your_api_key" \
     http://localhost:8080/admin/dlq/{id}/retry
```

---

#### Step 8 — Check Transaction List

```bash
curl -H "Authorization: Bearer your_api_key" \
     http://localhost:8080/transactions
```

Expected: list of processed transactions scoped to your merchant.

---

#### Common Test Errors & Fixes

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid or stale timestamp` | Hardcoded timestamp in body | Use `date -u` to generate dynamically |
| `invalid signature` | Wrong secret or body mismatch | Check `MOMO_WEBHOOK_SECRET` in `.env` — ensure no duplicates |
| `verification failed` | `transactionId` field missing or empty | Use `transactionId` not `financialTransactionId` |
| `missing signature header` | `X-Signature` header not sent | Add `-H "X-Signature: $SIG"` to curl |
| RequestBin shows nothing on retry | Replay protection working correctly | Expected — use a new `TXID` for each test |

---

### Go-Live Checklist

- [ ] Health endpoint returns `ok`
- [ ] Merchant registered and `api_key` saved
- [ ] Valid webhook accepted and delivered to RequestBin
- [ ] Replay protection confirmed (duplicate rejected silently)
- [ ] Stale timestamp rejected with 401
- [ ] Prometheus counters show non-zero values
- [ ] DLQ entry created after forced failure
- [ ] DLQ retry endpoint works
- [ ] Transaction list returns merchant-scoped data
- [ ] `.env` has no duplicate `MOMO_WEBHOOK_SECRET`
- [ ] MTN MoMo production credentials configured
- [ ] Webhook URL updated to production domain
- [ ] Monitoring (Prometheus + Grafana) connected

---

## Security Notes

- Never share your `api_key` or `MOMO_WEBHOOK_SECRET`
- Use HTTPS for all webhook endpoints in production
- Define `MOMO_WEBHOOK_SECRET` exactly once in `.env`
- Always generate fresh timestamps — never hardcode them
- Use `transactionId` as your idempotency key in your own systems
- Monitor DLQ regularly — failed deliveries mean missed payment notifications

---

## Support

- API docs: `http://localhost:8080/docs/index.html`
- Logs: `docker-compose logs paykit`
- Metrics: `http://localhost:8080/metrics/prometheus`

---

*Last updated: June 2026*