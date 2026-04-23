package storage

import (
	"context"
	"fmt"

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

// CreateMerchant inserts a new merchant and returns their ID
func (s *Store) CreateMerchant(m Merchant) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		context.Background(),
		`INSERT INTO merchants (name, api_key, webhook_url)
         VALUES ($1, $2, $3) RETURNING id`,
		m.Name, m.APIKey, m.WebhookURL,
	).Scan(&id)
	return id, err
}

// GetMerchantByAPIKey finds a merchant by their API key — used in middleware
func (s *Store) GetMerchantByAPIKey(apiKey string) (*Merchant, error) {
	m := &Merchant{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, name, api_key, webhook_url, active, created_at
         FROM merchants WHERE api_key = $1 AND active = true`,
		apiKey,
	).Scan(&m.ID, &m.Name, &m.APIKey, &m.WebhookURL, &m.Active, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetByExternalID finds a transaction by external_id scoped to a merchant
func (s *Store) GetByExternalID(externalID string, merchantID int64) (*Transaction, error) {
	transaction := &Transaction{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, merchant_id, provider_tx_id, external_id, amount, currency,
                status, received_at, created_at
         FROM transactions
         WHERE external_id = $1 AND merchant_id = $2
         ORDER BY created_at DESC LIMIT 1`,
		externalID, merchantID,
	).Scan(
		&transaction.ID, &transaction.MerchantID, &transaction.ProviderTxID, &transaction.ExternalID,
		&transaction.Amount, &transaction.Currency, &transaction.Status, &transaction.ReceivedAt, &transaction.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}

// ListTransactions returns filtered transactions scoped to a merchant
func (s *Store) ListTransactions(merchantID int64, status string, limit, offset int) ([]Transaction, error) {
	query := `SELECT id, merchant_id, provider_tx_id, external_id, amount, currency,
                     status, received_at, created_at
              FROM transactions
              WHERE merchant_id = $1`
	args := []any{merchantID}

	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}

	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var transaction Transaction
		if err := rows.Scan(
			&transaction.ID, &transaction.MerchantID, &transaction.ProviderTxID, &transaction.ExternalID,
			&transaction.Amount, &transaction.Currency, &transaction.Status, &transaction.ReceivedAt, &transaction.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}

func (s *Store) GetMerchantByID(id int64) (*Merchant, error) {
	m := &Merchant{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, name, api_key, webhook_url, active, created_at
         FROM merchants WHERE id = $1 AND active = true`,
		id,
	).Scan(&m.ID, &m.Name, &m.APIKey, &m.WebhookURL, &m.Active, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) InsertDeliveryLog(log DeliveryLog) error {
	_, err := s.db.Exec(
		context.Background(),
		`INSERT INTO delivery_logs
			(transaction_id, merchant_id, webhook_url, attempt, status, response_code, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		log.TransactionID,
		log.MerchantID,
		log.WebhookURL,
		log.Attempt,
		log.Status,
		log.ResponseCode,
		log.ErrorMessage,
	)
	return err
}

func (s *Store) GetDeliveryLogs(transactionID int64) ([]DeliveryLog, error) {
	rows, err := s.db.Query(
		context.Background(),
		`SELECT id, transaction_id, merchant_id, webhook_url, attempt, status, response_code, error_message, delivered_at
		 FROM delivery_logs
		WHERE transaction_id = $1 
		ORDER BY delivered_at DESC`,
		transactionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []DeliveryLog
	for rows.Next() {
		var log DeliveryLog
		if err := rows.Scan(
			&log.ID,
			&log.TransactionID,
			&log.MerchantID,
			&log.WebhookURL,
			&log.Attempt,
			&log.Status,
			&log.ResponseCode,
			&log.ErrorMessage,
			&log.DeliveredAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
