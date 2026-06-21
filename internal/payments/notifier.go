package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Brown-Moses/paykit/internal/metrics"
	"github.com/Brown-Moses/paykit/internal/storage"

	"github.com/Brown-Moses/paykit/pkg/momodto"
)

func NotifyMerchant(db *storage.Store, merchant *storage.Merchant, transaction *storage.Transaction) bool {
	if merchant.WebhookURL == "" {
		return false
	}
	if transaction.Status != storage.TxStatusSuccessful {
		return false
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
		return false
	}

	merchantIDLabel := strconv.FormatInt(merchant.ID, 10)

	// retry 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {

			wait := time.Duration(1<<attempt) * time.Second
			slog.Warn("notifier: retrying delivery", "attempt", attempt, "tx_id", transaction.ProviderTxID, "wait", wait)
			time.Sleep(wait)
		}

		deliveryLog := storage.DeliveryLog{
			TransactionID: transaction.ID,
			WebhookURL:    merchant.WebhookURL,
			Attempt:       attempt,
		}

		responseCode, err := postWebhook(merchant.WebhookURL, body)

		if err == nil {
			metrics.WebhookDeliveriesTotal.WithLabelValues(merchantIDLabel, string(storage.DeliveryStatusSuccess)).Inc()
		} else {
			// attempt 2 corresponds to final permanent failure, while attempts 0-1 are retrying
			statusLabel := string(storage.DeliveryStatusRetrying)
			if attempt == 2 {
				statusLabel = string(storage.DeliveryStatusFailed)
			}
			metrics.WebhookDeliveriesTotal.WithLabelValues(merchantIDLabel, statusLabel).Inc()
		}

		deliveryLog.ResponseCode = responseCode

		if err == nil {
			deliveryLog.Status = storage.DeliveryStatusSuccess
			db.InsertDeliveryLog(deliveryLog)
			slog.Info("notifier: delivered successfully", "url", merchant.WebhookURL, "tx_id", transaction.ProviderTxID, "attempt", attempt)
			return true
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

		// Enqueue DLQ on final permanent failure (do NOT delete, only mark resolved later)
		if attempt == 2 {
			dlq := storage.DeliveryDLQ{
				TransactionID:    transaction.ID,
				MerchantID:       merchant.ID,
				WebhookURL:       merchant.WebhookURL,
				AttemptCount:     attempt + 1,
				LastError:        err.Error(),
				LastResponseCode: deliveryLog.ResponseCode,
				Status:           storage.DLQStatusFailed,
				AvailableAt:      time.Now().UTC(),
			}

			_ = db.EnqueueDLQ(dlq)
			metrics.DLQEnqueuesTotal.WithLabelValues(merchantIDLabel, "delivery_failed").Inc()
		}

	}

	slog.Error("notifier: all retries exhausted", "tx_id", transaction.ProviderTxID)
	return false
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
