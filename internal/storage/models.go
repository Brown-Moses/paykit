package storage

import "time"

// HOLDS THE STATES
type TxStatus string

const (
	TxStatusPending    TxStatus = "PENDING"
	TxStatusSuccessful TxStatus = "SUCCESSFUL"
	TxStatusFailed     TxStatus = "FAILED"
)

// internal records
type Transaction struct {
	ID           int64
	MerchantID   int64
	ProviderTxID string
	ExternalID   string
	Amount       string
	Currency     string
	Status       TxStatus
	PayerMSISDN  string
	RawPayload   []byte
	ReceivedAt   time.Time
	CreatedAt    time.Time
}

type Merchant struct {
	ID         int64
	Name       string
	APIKey     string
	WebhookURL string
	Active     bool
	CreatedAt  time.Time
}
