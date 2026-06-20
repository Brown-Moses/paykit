package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/Brown-Moses/paykit/pkg/momodto"
)

func NotifyMerchant(db *storage.Store, merchant *storage.Merchant, transaction *storage.Transaction) {
	if merchant.WebhookURL == "" {
		return
	}
	if transaction.Status != storage.TxStatusSuccessful {
		return
	}

	payload := momodto.NotifyPayload{
		ExternalID: transaction.ExternalID,
		Status:     string(transaction.Status),
		Amount:     transaction.Amount,
		Currency:   transaction.Currency,
		Paid:       true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("notifier: failed to marshal payload", "tx_id", transaction.ProviderTxID, "error", err)
		return
	}

	// retry 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<attempt) * time.Second
			slog.Warn("notifier: retrying delivery", "attempt", attempt, "tx_id", transaction.ProviderTxID, "wait", wait)
			time.Sleep(wait)
		}

		deliveryLog := storage.DeliveryLog{
			TransactionID: transaction.ID,
			MerchantID:    merchant.ID,
			WebhookURL:    merchant.WebhookURL,
			Attempt:       attempt,
		}

		responseCode, err := postWebhook(merchant.WebhookURL, body)
		deliveryLog.ResponseCode = responseCode

		if err == nil {
			deliveryLog.Status = storage.DeliveryStatusSuccess
			db.InsertDeliveryLog(deliveryLog)
			slog.Info("notifier: delivered successfully", "url", merchant.WebhookURL, "tx_id", transaction.ProviderTxID, "attempt", attempt)
			return
		}

		// Last attempt (0-indexed: attempt 2 is the 3rd and final attempt)
		if attempt == 2 {
			deliveryLog.Status = storage.DeliveryStatusFailed
		} else {
			deliveryLog.Status = storage.DeliveryStatusRetrying
		}

		deliveryLog.ErrorMessage = err.Error()
		db.InsertDeliveryLog(deliveryLog)
		slog.Warn("notifier: attempt failed", "attempt", attempt, "tx_id", transaction.ProviderTxID, "error", err)
	}

	slog.Error("notifier: all retries exhausted", "tx_id", transaction.ProviderTxID)
}

func postWebhook(url string, body []byte) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, fmt.Errorf("merchant returned non-2xx: %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}
