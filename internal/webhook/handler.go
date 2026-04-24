package webhook

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"encoding/json"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/payments"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/Brown-Moses/paykit/pkg/momodto"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	verifier  *auth.Verifier
	store     *storage.Store
	startTime time.Time
}

func NewHandler(verifier *auth.Verifier, store *storage.Store, startTime time.Time) *Handler {
	return &Handler{
		verifier:  verifier,
		store:     store,
		startTime: startTime,
	}
}

// HandleMoMoWebhook godoc
// @Summary      Receive MTN MoMo payment webhook
// @Description  Receives and processes a payment webhook from MTN MoMo. Verifies HMAC-SHA256 signature, checks for replay attacks, stores the transaction, and notifies the merchant asynchronously.
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        X-Signature  header  string                true  "HMAC-SHA256 signature of request body"
// @Param        request      body    momodto.WebhookPayload true  "MTN MoMo webhook payload"
// @Success      200
// @Failure      400  {object}  object{error=string}
// @Failure      401  {object}  object{error=string}
// @Failure      422  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Router       /webhook/momo [post]

func (h *Handler) HandleMoMoWebhook(c *gin.Context) {
	// 1. Read raw body — needed for HMAC and storage
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read request body"})
		return
	}

	// 2. Decode into raw DTO
	var raw momodto.WebhookPayload
	if err := json.Unmarshal(body, &raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	// 3. Verify — HMAC + replay check
	signature := c.GetHeader("X-Signature")
	if err := h.verifier.Verify(body, signature, raw.TransactionID); err != nil {
		switch {
		case errors.Is(err, auth.ErrMissingSignature):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing signature header"})
		case errors.Is(err, auth.ErrInvalidSignature):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		case errors.Is(err, auth.ErrReplay):
			// Return 200 — MTN retries on anything else
			log.Printf("replay detected for tx %s — ignoring", raw.TransactionID)
			c.Status(http.StatusOK)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		}
		return
	}

	// 4. Parse into internal model
	transaction, err := payments.FromWebhook(raw, body)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	// 5. Save to Postgres
	if err := h.store.Insert(*transaction); err != nil {
		log.Printf("failed to insert transaction %s: %v", transaction.ProviderTxID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save transaction"})
		return
	}
	// 6. Fetch merchant by merchant_id on the transaction then notify async
	if transaction.MerchantID != 0 {
		merchant, err := h.store.GetMerchantByID(transaction.MerchantID)
		if err == nil {
			go payments.NotifyMerchant(h.store, merchant, transaction)
		} else {
			log.Printf("notifier: could not find merchant %d: %v", transaction.MerchantID, err)
		}
	}

	log.Printf("transaction %s received — external_id: %s amount: %s %s",
		transaction.ProviderTxID, transaction.ExternalID, transaction.Amount, transaction.Currency)

	c.Status(http.StatusOK)
}

// GetTransaction godoc
// @Summary      Get transaction by provider ID
// @Description  Returns a single transaction scoped to the authenticated merchant. The paid field gives a simple true/false answer.
// @Tags         Transactions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Provider transaction ID (e.g. TX-001)"
// @Success      200  {object}  object{paid=bool,status=string,provider_tx_id=string,external_id=string,amount=string,currency=string,received_at=string}
// @Failure      401  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Router       /transactions/{id} [get]
func (h *Handler) GetTransaction(c *gin.Context) {
	merchant := auth.MerchantFrom(c)
	providerTxID := c.Param("id")

	transaction, err := h.store.GetByID(providerTxID)
	if err != nil || transaction.MerchantID != merchant.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(http.StatusOK, normalizeTransaction(transaction))

}

// ListTransactions godoc
// @Summary      List and filter transactions
// @Description  Returns paginated transactions for the authenticated merchant. Filter by status or look up a specific order using external_id.
// @Tags         Transactions
// @Produce      json
// @Security     BearerAuth
// @Param        external_id  query     string  false  "Look up by order ID (e.g. order_987)"
// @Param        status       query     string  false  "Filter by status: SUCCESSFUL, FAILED, PENDING"
// @Param        page         query     int     false  "Page number (default: 1)"
// @Param        limit        query     int     false  "Results per page (default: 20)"
// @Success      200          {object}  object{page=int,limit=int,count=int,data=array}
// @Failure      401          {object}  object{error=string}
// @Failure      500          {object}  object{error=string}
// @Router       /transactions [get]
func (h *Handler) ListTransactions(c *gin.Context) {
	merchant := auth.MerchantFrom(c)

	status := c.Query("status")
	externalID := c.Query("external_id")

	// Single lookup by external_id
	if externalID != "" {
		transaction, err := h.store.GetByExternalID(externalID, merchant.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
			return
		}
		c.JSON(http.StatusOK, normalizeTransaction(transaction))
		return
	}

	// Paginated list
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := (page - 1) * limit

	transactions, err := h.store.ListTransactions(merchant.ID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch transactions"})
		return
	}

	results := make([]gin.H, len(transactions))
	for i, tx := range transactions {
		results[i] = normalizeTransaction(&tx)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"limit": limit,
		"count": len(results),
		"data":  results,
	})
}

// GetDeliveryLogs godoc
// @Summary      Get delivery attempts for a transaction
// @Description  Returns all webhook delivery attempts made to the merchant's webhook URL for a given transaction. Shows attempt number, status, HTTP response code, and any error messages.
// @Tags         Delivery Logs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Provider transaction ID (e.g. TX-001)"
// @Success      200  {object}  object{transaction_id=string,webhook_url=string,attempts=int,logs=array}
// @Failure      401  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Failure      500  {object}  object{error=string}
// @Router       /transactions/{id}/deliveries [get]
func (h *Handler) GetDeliveryLogs(c *gin.Context) {
	merchant := auth.MerchantFrom(c)
	providerTxID := c.Param("id")

	// First verify transaction belongs to this merchant
	transaction, err := h.store.GetByID(providerTxID)
	if err != nil || transaction.MerchantID != merchant.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	logs, err := h.store.GetDeliveryLogs(transaction.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch delivery logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": providerTxID,
		"webhook_url":    merchant.WebhookURL,
		"attempts":       len(logs),
		"logs":           logs,
	})
}

// Health godoc
// @Summary      Health check
// @Description  Returns server and database status. Use this to verify the service is running before sending webhooks.
// @Tags         System
// @Produce      json
// @Success      200  {object}  object{status=string,database=string}
// @Failure      503  {object}  object{status=string,database=string}
// @Router       /health [get]
func (h *Handler) Health(c *gin.Context) {
	uptime := time.Since(h.startTime).Round(time.Second).String()

	if err := h.store.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "degraded",
			"database": "unreachable",
			"version":  "1.0.0",
			"uptime":   uptime,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"database": "connected",
		"version":  "1.0.0",
		"uptime":   uptime,
	})
}

// normalizeTransaction returns a clean response — not raw DB fields
func normalizeTransaction(tx *storage.Transaction) gin.H {
	return gin.H{
		"paid":           tx.Status == storage.TxStatusSuccessful,
		"status":         tx.Status,
		"provider_tx_id": tx.ProviderTxID,
		"external_id":    tx.ExternalID,
		"amount":         tx.Amount,
		"currency":       tx.Currency,
		"received_at":    tx.ReceivedAt,
	}
}
