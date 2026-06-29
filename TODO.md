# PayKit Roadmap Implementation Tracker

## Step 1: Timestamp validation (Done)
- [x] Reject invalid/stale MTN webhook timestamps (401)
- [x] Updated tests to match new behavior

## Step 2: Dead letter queue (DLQ) for failed merchant webhook deliveries (Done)
- [x] Add DLQ table to database schema
- [x] Add DLQ model + database store methods
- [x] Enqueue into DLQ after delivery retries are exhausted in the notifier service
- [x] Add admin REST endpoints to inspect and manually retry DLQ items

## Step 3: Prometheus & Merchant Metrics (Done)
- [x] Add Prometheus exposition format scrape endpoint `/metrics/prometheus`
- [x] Register custom counters & gauges (WebhookDeliveriesTotal, DLQEnqueuesTotal, DLQRetriesTotal, DLQItemsResolvedTotal)
- [x] Wire Prometheus handler and registration during server initialization
- [x] Implement merchant-scoped metrics endpoint `/metrics` protected by API key auth

## Step 4: Per-merchant rate limiting (Next)
- [ ] Implement token bucket middleware (e.g. using `didip/tollbooth` or a custom Gin middleware)
- [ ] Support configuration via environment variables

## Step 5: Idempotency at HTTP layer (Later)
- [ ] Add Idempotency-Key header checking and/or request deduplication cache

## Step 6: Multi-tenancy restructure (Later)
- [ ] Enforce database tenant isolation scoping across all queries

## Step 7-9: Business items as docs/config (Later)
- [ ] Document MTN API versioning strategy
- [ ] Design pull-based license auditing mechanisms
- [ ] Draft Rwanda vs Uganda launch market assessment report
