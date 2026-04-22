package webhook

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"encoding/json"

	"github.com/Brown-Moses/paykit/internal/auth"
	"github.com/Brown-Moses/paykit/internal/payments"
	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/Brown-Moses/paykit/pkg/momodto"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	verifier *auth.Verifier
	store    *storage.Store
}

func NewHandler(verifier *auth.Verifier, store *storage.Store) *Handler {
	return &Handler{
		verifier: verifier,
		store:    store,
	}
}

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
			go payments.NotifyMerchant(merchant, transaction)
		} else {
			log.Printf("notifier: could not find merchant %d: %v", transaction.MerchantID, err)
		}
	}

	log.Printf("transaction %s received — external_id: %s amount: %s %s",
		transaction.ProviderTxID, transaction.ExternalID, transaction.Amount, transaction.Currency)

	c.Status(http.StatusOK)
}

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
