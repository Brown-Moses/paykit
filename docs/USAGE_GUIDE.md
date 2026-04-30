# PayKit Usage Guide

## Overview

PayKit is a production-ready, multi-tenant payment notification engine that bridges MTN MoMo payments to your business systems. This guide covers everything you need to know to operate PayKit and integrate it with your applications.

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
5. [Testing](#testing)
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

### Configuration

#### Required Environment Variables

```bash
# Database connection (required)
DATABASE_URL=postgres://paykit:paykit_secret@host:5432/paykit?sslmode=require

# MTN MoMo webhook secret (required)
MOMO_WEBHOOK_SECRET=your_shared_secret_from_mtn_portal

# Server port (optional, defaults to 8080)
PORT=8080
```

#### Optional Security

```bash
# IP whitelisting for webhooks (recommended for production)
ALLOWED_IPS=196.47.12.0/24,196.47.13.0/24
```

#### Database Setup

PayKit uses PostgreSQL 15+. Create database and user:

```sql
CREATE DATABASE paykit;
CREATE USER paykit WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE paykit TO paykit;
```

Run migrations:
```bash
make migrate
```

### Monitoring & Maintenance

#### Health Checks

```bash
# Check service health
curl http://your-paykit.com/health

# Check database connectivity
make ping-db
```

#### Key Metrics to Monitor

- **Webhook Processing Rate**: Transactions per minute
- **Delivery Success Rate**: Percentage of successful merchant notifications
- **Response Times**: P95 webhook processing time
- **Error Rates**: Failed webhook validations

#### Database Maintenance

```bash
# Backup database
make db-backup

# Restore from backup
make db-restore FILE=backups/paykit_20240101.sql

# Monitor table sizes
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### Log Analysis

PayKit logs all webhook deliveries. Check delivery success:

```sql
SELECT
    status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM delivery_logs
WHERE delivered_at >= NOW() - INTERVAL '24 hours'
GROUP BY status;
```

### Troubleshooting

#### Common Issues

**Webhooks not being received:**
- Check MTN MoMo configuration points to correct URL
- Verify `MOMO_WEBHOOK_SECRET` matches MTN portal
- Check firewall allows incoming connections

**Merchant notifications failing:**
- Verify merchant `webhook_url` is accessible
- Check for network timeouts (PayKit retries automatically)
- Review delivery_logs for specific error messages

**Database connection issues:**
- Verify `DATABASE_URL` is correct
- Check PostgreSQL is running
- Ensure SSL settings match database configuration

**High memory usage:**
- Monitor goroutine count (PayKit uses goroutines for async notifications)
- Check for memory leaks in custom code
- Consider increasing server resources

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

# Check merchant registration
SELECT id, name, webhook_url, active
FROM merchants;
```

---

## For Merchants (Customers)

### Merchant Registration

Register your business with PayKit:

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
  "merchant_id": 123,
  "api_key": "pk_live_abc123def456",
  "webhook_url": "https://your-app.com/webhooks/paykit"
}
```

**Important:** Save your `api_key` securely - it's required for all API calls.

### MTN MoMo Setup

1. **Get MTN MoMo Credentials:**
   - Sign up at MTN Developer Portal
   - Get your `webhook_secret` (shared with PayKit)
   - Note your subscriber ID and API key

2. **Configure Webhook URL:**
   - In MTN portal, set webhook URL to:
     ```
     https://your-paykit.com/webhook/momo/123
     ```
     (Replace 123 with your merchant_id)

3. **Test in Sandbox:**
   - Use MTN sandbox environment first
   - Test with small amounts
   - Verify webhook delivery

### Webhook Integration

PayKit sends notifications to your `webhook_url` when payments are received.

#### Notification Payload

```json
{
  "event_type": "payment.successful",
  "merchant_id": 123,
  "transaction": {
    "provider_tx_id": "TX-123456789",
    "external_id": "ORDER-ABC-123",
    "amount": 5000.00,
    "currency": "RWF",
    "status": "SUCCESSFUL",
    "received_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Handle the notification:

```javascript
// Example Node.js handler
app.post('/webhooks/paykit', (req, res) => {
  const { event_type, transaction } = req.body;

  if (event_type === 'payment.successful') {
    // Update your order status
    await updateOrder(transaction.external_id, 'paid');

    // Fulfill the order
    await fulfillOrder(transaction.external_id);
  }

  res.status(200).send('OK');
});
```

```python
# Example Python handler
@app.route('/webhooks/paykit', methods=['POST'])
def paykit_webhook():
    data = request.get_json()

    if data['event_type'] == 'payment.successful':
        # Update order status
        update_order(data['transaction']['external_id'], 'paid')

        # Send confirmation email
        send_payment_confirmation(data['transaction'])

    return 'OK', 200
```

#### Security Best Practices

- **Verify signatures** (if PayKit adds them in future)
- **Use HTTPS** for your webhook endpoint
- **Implement idempotency** using `provider_tx_id`
- **Respond quickly** (within 5 seconds)
- **Return 200 OK** for successful processing

### API Usage

Use your `api_key` for authenticated requests:

```bash
# All API calls need this header
Authorization: Bearer pk_live_abc123def456
```

#### Query Transactions

```bash
# Get all transactions
curl -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/transactions

# Filter by status
curl -H "Authorization: Bearer pk_live_abc123def456" \
     "http://your-paykit.com/transactions?status=SUCCESSFUL"

# Get specific transaction
curl -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/transactions/TX-123456789
```

#### Check Delivery Status

```bash
# Get delivery attempts for a transaction
curl -H "Authorization: Bearer pk_live_abc123def456" \
     http://your-paykit.com/transactions/TX-123456789/deliveries
```

### Testing

#### Using the Postman Collection

1. Import `demo/postman/paykit.postman_collection.json`
2. Set variables:
   - `base_url`: `http://localhost:8080` (or your PayKit URL)
   - `webhook_secret`: Your test secret
   - `api_key`: Your merchant API key
3. Run the collection in order

#### Manual Testing

```bash
# 1. Register merchant
curl -X POST http://localhost:8080/merchants \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Merchant", "webhook_url": "https://httpbin.org/post"}'

# 2. Simulate MTN webhook (requires proper signature)
# Use the Postman collection for proper signature calculation
```

#### Test Webhook Endpoint

```bash
# Test your webhook handler
curl -X POST https://your-app.com/webhooks/paykit \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "payment.successful",
    "transaction": {
      "provider_tx_id": "TEST-123",
      "external_id": "ORDER-TEST-001",
      "amount": 1000.00,
      "currency": "RWF",
      "status": "SUCCESSFUL"
    }
  }'
```

### Go-Live Checklist

- [ ] Merchant account registered with PayKit
- [ ] MTN MoMo sandbox testing completed
- [ ] Webhook endpoint implemented and tested
- [ ] Idempotency handling implemented
- [ ] Error handling and logging in place
- [ ] MTN MoMo production credentials obtained
- [ ] PayKit production instance configured
- [ ] Webhook URL updated to production
- [ ] Small test transaction processed successfully
- [ ] Monitoring and alerting set up
- [ ] Support contact information documented

---

## Support

### For Operators
- Check logs: `docker-compose logs paykit`
- Database queries in `/internal/storage/migrate.sql`
- API documentation at `/docs/index.html`

### For Merchants
- API documentation at `https://your-paykit.com/docs/index.html`
- Check transaction status via API
- Review delivery logs for webhook issues

### Common Questions

**Q: How do I handle duplicate webhooks?**
A: PayKit prevents duplicates using `provider_tx_id`. Implement idempotency in your webhook handler.

**Q: What happens if my webhook endpoint is down?**
A: PayKit retries up to 3 times with exponential backoff. Check delivery logs for status.

**Q: Can I change my webhook URL?**
A: Contact PayKit operator to update your merchant record.

**Q: How do I test without real MTN payments?**
A: Use the Postman collection with test signatures, or MTN sandbox environment.

---

## Security Notes

- Never share your `api_key` or `MOMO_WEBHOOK_SECRET`
- Use HTTPS for all webhook endpoints
- Implement proper input validation
- Monitor for unusual activity
- Keep PayKit updated with latest security patches

---

*Last updated: April 2026*</content>
<parameter name="filePath">/home/brown-moses/go/Go/paykitt/paykit/USAGE_GUIDE.md