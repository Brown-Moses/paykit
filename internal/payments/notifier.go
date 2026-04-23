package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
		log.Println("notifier: failed to marshal payload for %s: %v", transaction.ProviderTxID, err)
		return
	}

	//retry 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<attempt) * time.Second
			log.Printf("notifier: retry %d for %s - waiting %s", attempt, transaction.ProviderTxID, wait)
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
			log.Printf("notifier: delivered to %s for transaction %s on attempt %d",
				merchant.WebhookURL, transaction.ProviderTxID, attempt)
			return
		}

		if err == nil {
			log.Printf("notifier: delivered to %s for tx %s", merchant.WebhookURL, transaction.ProviderTxID)
			return
		}
		// Last attempt
		if attempt == 3 {
			deliveryLog.Status = storage.DeliveryStatusFailed
		} else {
			deliveryLog.Status = storage.DeliveryStatusRetrying
		}

		deliveryLog.ErrorMessage = err.Error()
		db.InsertDeliveryLog(deliveryLog)
		log.Printf("notifier: attempt %d failed for %s: %v", attempt, transaction.ProviderTxID, err)
	}

	log.Printf("notifier: all retries exhausted for transaction %s", transaction.ProviderTxID)
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
