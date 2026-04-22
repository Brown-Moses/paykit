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
