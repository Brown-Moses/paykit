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

type NotifyPayload struct {
	ExternalID string `json:"external_id"`
	Status     string `json:"status"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Paid       bool   `json:"paid"`
}

type DeliveryStatus string

const (
	DeliveryStatusSuccess  DeliveryStatus = "SUCCESS"
	DeliveryStatusFailed   DeliveryStatus = "FAILED"
	DeliveryStatusRetrying DeliveryStatus = "RETRYING"
)

type DeliveryLog struct {
	ID            int64
	TransactionID int64
	MerchantID    int64
	WebhookURL    string
	Attempt       int
	Status        DeliveryStatus
	ResponseCode  int
	ErrorMessage  string
	DeliveredAt   time.Time
}
