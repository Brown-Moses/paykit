package webhook

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/metrics"

	"github.com/Brown-Moses/paykit/internal/payments"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/gin-gonic/gin"
)

// ListDLQ godoc
// @Summary      List DLQ records
// @Description  Returns delivery DLQ entries (failed merchant webhook deliveries) for the authenticated merchant.
// @Tags         Admin DLQ
// @Produce      json
// @Security     BearerAuth
// @Param        page         query     int     false  "Page number (default: 1)"
// @Param        limit        query     int     false  "Results per page (default: 20)"
// @Success      200  {object}  object{page=int,limit=int,count=int,data=array}
// @Failure      401  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Router       /admin/dlq [get]
func (h *Handler) ListDLQ(c *gin.Context) {
	merchant := auth.MerchantFrom(c)

	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	offset := (page - 1) * limit

	items, err := h.store.ListDLQ(merchant.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch dlq"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"limit": limit,
		"count": len(items),
		"data":  items,
	})
}

// RetryDLQ godoc
// @Summary      Retry a DLQ item immediately
// @Description  Triggers immediate retry for a permanently failed merchant webhook delivery. On success it resolves the DLQ record.
// @Tags         Admin DLQ
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "DLQ id"
// @Success      200  {object}  object{status=string}
// @Failure      401  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Router       /admin/dlq/{id}/retry [post]
func (h *Handler) RetryDLQ(c *gin.Context) {
	merchant := auth.MerchantFrom(c)
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dlq id"})
		return
	}

	d, err := h.store.GetDLQ(id, merchant.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dlq item not found"})
		return
	}
	if d.Status == storage.DLQStatusResolved {
		c.JSON(http.StatusOK, gin.H{"status": "already resolved"})
		return
	}

	// Mark the DLQ item as re-queued so it doesn't get retried concurrently / multiple times.
	if err := h.store.MarkDLQRequeued(d.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not requeue dlq"})
		return
	}

	// Load merchant + transaction and re-run the same notifier delivery logic.

	m, err := h.store.GetMerchantByID(merchant.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load merchant"})
		return
	}

	tx, err := h.store.GetTransactionInternal(d.TransactionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	// Note: NotifyMerchant will perform the actual merchant delivery retries and always write to delivery_logs.
	// We intentionally do NOT create a new DLQ record on retry to avoid duplicate clutter.
	// Resolving the DLQ record on success would require checking delivery_logs outcomes and is not implemented in this MVP.

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("dlq retry: recovered", "panic", r)
			}
		}()

		// NotifyMerchant returns delivery outcome so we can auto-resolve DLQ on success.
		merchantIDLabel := strconv.FormatInt(m.ID, 10)

		metrics.DLQRetriesTotal.WithLabelValues(merchantIDLabel, "success").Inc()
		if ok := payments.NotifyMerchant(h.store, m, tx); ok {
			_ = h.store.ResolveDLQ(d.ID, time.Now().UTC())
			metrics.DLQItemsResolvedTotal.WithLabelValues(merchantIDLabel).Inc()
		} else {
			metrics.DLQRetriesTotal.WithLabelValues(merchantIDLabel, "failure").Inc()
		}

	}()

	c.JSON(http.StatusOK, gin.H{"status": "retry triggered"})
}
