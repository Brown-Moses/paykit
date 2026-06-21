package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Delivery success/failure for merchant webhook notifications
	WebhookDeliveriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paykit_merchant_webhook_deliveries_total",
			Help: "Total number of merchant webhook delivery attempts.",
		},
		[]string{"merchant_id", "status"},
	)

	// DLQ events
	DLQEnqueuesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paykit_delivery_dlq_enqueues_total",
			Help: "Total number of delivery attempts enqueued into DLQ.",
		},
		[]string{"merchant_id", "reason"},
	)

	DLQRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paykit_delivery_dlq_retries_total",
			Help: "Total number of DLQ retry attempts.",
		},
		[]string{"merchant_id", "result"},
	)

	DLQItemsResolvedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paykit_delivery_dlq_items_resolved_total",
			Help: "Total number of DLQ items resolved after successful retry.",
		},
		[]string{"merchant_id"},
	)
)

func Register() {
	prometheus.MustRegister(
		WebhookDeliveriesTotal,
		DLQEnqueuesTotal,
		DLQRetriesTotal,
		DLQItemsResolvedTotal,
	)
}
