package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Exists(providerTxID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM transactions WHERE provider_tx_id = $1)`,
		providerTxID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) Insert(transaction Transaction) error {
	_, err := s.db.Exec(
		context.Background(),
		`INSERT INTO transactions
            (provider_tx_id, external_id, amount, currency, status, payer_msisdn, raw_payload, received_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		transaction.ProviderTxID,
		transaction.ExternalID,
		transaction.Amount,
		transaction.Currency,
		transaction.Status,
		transaction.PayerMSISDN,
		transaction.RawPayload,
		transaction.ReceivedAt,
	)
	return err
}

func (s *Store) GetByID(providerTxID string) (*Transaction, error) {
	transaction := &Transaction{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, provider_tx_id, external_id, amount, currency, status, received_at, created_at
         FROM transactions WHERE provider_tx_id = $1`,
		providerTxID,
	).Scan(
		&transaction.ID, &transaction.ProviderTxID, &transaction.ExternalID, &transaction.Amount,
		&transaction.Currency, &transaction.Status, &transaction.ReceivedAt, &transaction.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}
