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

func NotifyMerchant(merchant *storage.Merchant, transaction *storage.Transaction) {
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

		err := postWebhook(merchant.WebhookURL, body)
		if err == nil {
			log.Printf("notifier: delivered to %s for tx %s", merchant.WebhookURL, transaction.ProviderTxID)
			return
		}

		log.Printf("notifier: attempt %d failed for %s: %v", attempt+1, transaction.ProviderTxID, err)
	}

	log.Printf("notifier: all retries exhausted for transaction %s — merchant not notified", transaction.ProviderTxID)
}

func postWebhook(url string, body []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("merchant returned non-2xx: %d", resp.StatusCode)
	}

	return nil
}
