DLQ design notes (for implementation step 2)

Entity: delivery_dlq
- id BIGSERIAL PK
- transaction_id BIGINT NOT NULL REFERENCES transactions(id)
- merchant_id BIGINT NOT NULL REFERENCES merchants(id)
- webhook_url TEXT NOT NULL
- attempt_count INT NOT NULL
- last_error TEXT
- last_response_code INT
- created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
- available_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
- status TEXT NOT NULL DEFAULT 'PENDING' (or ENUM)

Store methods
- EnqueueDLQ(log DeliveryDLQ)
- ListDLQ(merchantID, limit, offset)
- GetDLQ(id)
- MarkDLQRequeued(id)
- DeleteDLQ(id) after successful requeue

Notifier integration
- On final retry (attempt==2) where err!=nil: insert into delivery_dlq in addition to delivery_logs.
- Ensure DLQ insertion does not block webhook handler; notifier is already async.

Admin endpoints
- GET /admin/dlq (BearerAuth) returns DLQ records for authenticated merchant
- POST /admin/dlq/:id/retry (BearerAuth) triggers immediate retry logic for that transaction.

Routing
- Add admin group protected by RequireAPIKey middleware.

Implementation choice
- Minimal viable DLQ: table + enqueue + merchant-scoped listing + retry endpoint.

