package payments

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/Brown-Moses/paykit/internal/storage"
	"github.com/Brown-Moses/paykit/pkg/momodto"
)

func FromWebhook(p momodto.WebhookPayload, rawBody []byte) (*storage.Transaction, error) {

	if p.TransactionID == "" {
		return nil, fmt.Errorf("missing transactionId")
	}

	if p.ExternalId == "" {
		return nil, fmt.Errorf("missing externalId - cannot match to an order")
	}

	if p.Amount == "" {
		return nil, fmt.Errorf("missing amount")
	}

	timeStamp, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: must be RFC3339")
	}


	return &storage.Transaction{
		ProviderTxID: p.TransactionID,
		ExternalID:   p.ExternalId,
		Amount:       p.Amount,
		Currency:     p.Currency,
		Status:       storage.TxStatus(p.Status),
		PayerMSISDN:  hashMSISDN(p.Payer.PartyID),
		RawPayload:   rawBody,
		ReceivedAt:   timeStamp,
	}, nil
}

// MSISDN — Mobile Station Integrated Services Digital Network number
func hashMSISDN(msisdn string) string {
	h := sha256.Sum256([]byte(msisdn))
	return fmt.Sprintf("%x", h)
}
