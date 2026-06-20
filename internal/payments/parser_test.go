package payments

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/Brown-Moses/paykit/pkg/momodto"
)

func TestFromWebhook(t *testing.T) {
	rawBody := []byte(`{"transactionId":"TX-001"}`)
	validPayload := momodto.WebhookPayload{
		TransactionID: "TX-001",
		ExternalId:    "order_999",
		Amount:        "1500.00",
		Currency:      "RWF",
		Status:        "SUCCESSFUL",
		Payer: momodto.Payer{
			PartyIDType: "MSISDN",
			PartyID:     "250788123456",
		},
		Timestamp: "2026-06-20T12:00:00Z",
	}

	t.Run("Valid Payload", func(t *testing.T) {
		tx, err := FromWebhook(validPayload, rawBody)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tx.ProviderTxID != "TX-001" {
			t.Errorf("expected ProviderTxID TX-001, got %s", tx.ProviderTxID)
		}
		if tx.ExternalID != "order_999" {
			t.Errorf("expected ExternalID order_999, got %s", tx.ExternalID)
		}
		if tx.Amount != "1500.00" {
			t.Errorf("expected Amount 1500.00, got %s", tx.Amount)
		}
		if tx.Currency != "RWF" {
			t.Errorf("expected Currency RWF, got %s", tx.Currency)
		}
		if string(tx.Status) != "SUCCESSFUL" {
			t.Errorf("expected Status SUCCESSFUL, got %s", tx.Status)
		}

		// Verify MSISDN hash
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte("250788123456")))
		if tx.PayerMSISDN != expectedHash {
			t.Errorf("expected hashed phone %s, got %s", expectedHash, tx.PayerMSISDN)
		}

		// Verify timestamp parsing
		expectedTime, _ := time.Parse(time.RFC3339, "2026-06-20T12:00:00Z")
		if !tx.ReceivedAt.Equal(expectedTime) {
			t.Errorf("expected ReceivedAt %v, got %v", expectedTime, tx.ReceivedAt)
		}
	})

	t.Run("Missing Transaction ID", func(t *testing.T) {
		p := validPayload
		p.TransactionID = ""
		_, err := FromWebhook(p, rawBody)
		if err == nil {
			t.Error("expected error for missing transactionId, got nil")
		}
	})

	t.Run("Missing External ID", func(t *testing.T) {
		p := validPayload
		p.ExternalId = ""
		_, err := FromWebhook(p, rawBody)
		if err == nil {
			t.Error("expected error for missing externalId, got nil")
		}
	})

	t.Run("Missing Amount", func(t *testing.T) {
		p := validPayload
		p.Amount = ""
		_, err := FromWebhook(p, rawBody)
		if err == nil {
			t.Error("expected error for missing amount, got nil")
		}
	})

	t.Run("Invalid Timestamp Fallback", func(t *testing.T) {
		p := validPayload
		p.Timestamp = "invalid-date"
		before := time.Now().UTC()
		tx, err := FromWebhook(p, rawBody)
		after := time.Now().UTC()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tx.ReceivedAt.Before(before) || tx.ReceivedAt.After(after) {
			t.Errorf("expected fallback timestamp to be roughly now (%v to %v), got %v", before, after, tx.ReceivedAt)
		}
	})
}
