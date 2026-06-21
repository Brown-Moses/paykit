# TODO

- [x] Step 1: Add `/metrics/prometheus` route in `api/routes.go` using `internal/metrics.PrometheusHandler()`. 
- [x] Step 2: Call `metrics.Register()` during startup in `cmd/paykit/main.go` before creating the server/router.

- [x] Step 3: Wire `WebhookDeliveriesTotal` and `DLQEnqueuesTotal` increments in `internal/payments/notifier.go`.
- [x] Step 4: Wire `DLQRetriesTotal` and `DLQItemsResolvedTotal` increments in `internal/webhook/dlq_admin.go`.
- [ ] Step 5: Run `go test ./...` and do a manual curl scrape of `/metrics/prometheus` to confirm counters increment.

# PayKit Roadmap Implementation Tracker
# TODO
## Step 1: Timestamp validation (done)
- [x] Reject invalid/stale MTN webhook timestamps (401)
- [x] Updated tests to match new behavior
## Step 2: Dead letter queue (DLQ) for failed merchant webhook deliveries (in progress)
- [x] Add DLQ table to schema
- [x] Add DLQ model + store methods
- [x] Enqueue into DLQ after retries exhausted in notifier
- [x] Add admin endpoints to inspect/retry DLQ
## Step 3: Prometheus metrics (next)
- [ ] Add Prometheus exposition format endpoint (or /metrics/prometheus)
- [ ] Add required counters/gauges
- [ ] Wire into router + swagger
- [x] (Done in this iteration) Confirmed DLQ status/migration, auto-resolve on retry, and merchant-scoped metrics.
## Step 4: Per-merchant rate limiting (later)
- [ ] Middleware + storage (in-memory/redis)
## Step 5: Idempotency at HTTP layer (later)
- [ ] Add Idempotency-Key handling and/or request dedupe cache
## Step 6: Multi-tenancy restructure (later)
- [ ] Decide tenant model + enforce tenant scoping across queries
## Step 7-9: Business items as docs/config (later)
- [ ] MTN API versioning strategy
- [ ] Pull-based license audit design
- [ ] Rwanda vs Uganda first-market recommendation
- [ ] Step 1: Add `/metrics/prometheus` route in `api/routes.go` using `internal/metrics.PrometheusHandler()`.
- [ ] Step 2: Call `metrics.Register()` during startup in `cmd/paykit/main.go` before creating the server/router.
- [ ] Step 3: Wire `WebhookDeliveriesTotal` and `DLQEnqueuesTotal` increments in `internal/payments/notifier.go`.
- [ ] Step 4: Wire `DLQRetriesTotal` and `DLQItemsResolvedTotal` increments in `internal/webhook/dlq_admin.go`.
- [ ] Step 5: Run `go test ./...` and do a manual curl scrape of `/metrics/prometheus` to confirm counters increment.

