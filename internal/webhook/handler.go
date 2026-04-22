package webhook

import (
	"errors"
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

	log.Printf("transaction %s received — external_id: %s amount: %s %s",
		transaction.ProviderTxID, transaction.ExternalID, transaction.Amount, transaction.Currency)

	c.Status(http.StatusOK)
}
