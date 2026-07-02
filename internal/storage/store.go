package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Ping checks the database connection
func (s *Store) Ping() error {
	return s.db.Ping(context.Background())
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
            (provider_tx_id, external_id, amount, currency, status, payer_msisdn, raw_payload, received_at, merchant_id)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		transaction.ProviderTxID,
		transaction.ExternalID,
		transaction.Amount,
		transaction.Currency,
		transaction.Status,
		transaction.PayerMSISDN,
		transaction.RawPayload,
		transaction.ReceivedAt,
		transaction.MerchantID,
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
	hashedKey := hashAPIKey(m.APIKey)
	if m.PlanType == "" {
		m.PlanType = "free"
	}
	if m.MaxMonthlyCalls == 0 {
		m.MaxMonthlyCalls = 1000
	}
	var id int64
	err := s.db.QueryRow(
		context.Background(),
		`INSERT INTO merchants (name, api_key, webhook_url, plan_type, max_monthly_calls, current_month_calls)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		m.Name, hashedKey, m.WebhookURL, m.PlanType, m.MaxMonthlyCalls, m.CurrentMonthCalls,
	).Scan(&id)
	return id, err
}

// GetMerchantByAPIKey finds a merchant by their API key — used in middleware
func (s *Store) GetMerchantByAPIKey(apiKey string) (*Merchant, error) {
	hashedKey := hashAPIKey(apiKey)
	m := &Merchant{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, name, api_key, webhook_url, active, plan_type, max_monthly_calls, current_month_calls, created_at
         FROM merchants WHERE api_key = $1 AND active = true`,
		hashedKey,
	).Scan(&m.ID, &m.Name, &m.APIKey, &m.WebhookURL, &m.Active, &m.PlanType, &m.MaxMonthlyCalls, &m.CurrentMonthCalls, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
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
		`SELECT id, name, api_key, webhook_url, active, plan_type, max_monthly_calls, current_month_calls, created_at
         FROM merchants WHERE id = $1 AND active = true`,
		id,
	).Scan(&m.ID, &m.Name, &m.APIKey, &m.WebhookURL, &m.Active, &m.PlanType, &m.MaxMonthlyCalls, &m.CurrentMonthCalls, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// IncrementMerchantCallCount increments the call usage counter for a merchant
func (s *Store) IncrementMerchantCallCount(id int64) error {
	_, err := s.db.Exec(
		context.Background(),
		`UPDATE merchants SET current_month_calls = current_month_calls + 1 WHERE id = $1`,
		id,
	)
	return err
}

// UpdateMerchantWebhookURL updates the webhook URL for a merchant
func (s *Store) UpdateMerchantWebhookURL(id int64, webhookURL string) error {
	_, err := s.db.Exec(
		context.Background(),
		`UPDATE merchants SET webhook_url = $1 WHERE id = $2`,
		webhookURL,
		id,
	)
	return err
}

// ListAllMerchants returns all merchants in the system (for operator dashboard)
func (s *Store) ListAllMerchants() ([]Merchant, error) {
	rows, err := s.db.Query(
		context.Background(),
		`SELECT id, name, api_key, webhook_url, active, plan_type, max_monthly_calls, current_month_calls, created_at
		 FROM merchants
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants []Merchant
	for rows.Next() {
		var m Merchant
		if err := rows.Scan(
			&m.ID, &m.Name, &m.APIKey, &m.WebhookURL, &m.Active,
			&m.PlanType, &m.MaxMonthlyCalls, &m.CurrentMonthCalls, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		merchants = append(merchants, m)
	}
	return merchants, nil
}

// SetMerchantActive activates or deactivates a merchant
func (s *Store) SetMerchantActive(id int64, active bool) error {
	_, err := s.db.Exec(
		context.Background(),
		`UPDATE merchants SET active = $1 WHERE id = $2`,
		active,
		id,
	)
	return err
}

// UpdateMerchantQuota updates the plan type and maximum monthly calls limit for a merchant
func (s *Store) UpdateMerchantQuota(id int64, planType string, maxCalls int) error {
	_, err := s.db.Exec(
		context.Background(),
		`UPDATE merchants SET plan_type = $1, max_monthly_calls = $2 WHERE id = $3`,
		planType,
		maxCalls,
		id,
	)
	return err
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

func (s *Store) EnqueueDLQ(dlq DeliveryDLQ) error {
	_, err := s.db.Exec(
		context.Background(),
		`INSERT INTO delivery_dlq (transaction_id, merchant_id, webhook_url, attempt_count, last_error, last_response_code, status, available_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, NOW())`,
		dlq.TransactionID,
		dlq.MerchantID,
		dlq.WebhookURL,
		dlq.AttemptCount,
		dlq.LastError,
		dlq.LastResponseCode,
		dlq.Status,
		dlq.AvailableAt,
	)
	return err
}

func (s *Store) ResolveDLQ(id int64, resolvedAt time.Time) error {
	_, err := s.db.Exec(
		context.Background(),
		`UPDATE delivery_dlq
		 SET status = $1, resolved_at = $2
		 WHERE id = $3`,
		DLQStatusResolved,
		resolvedAt,
		id,
	)
	return err
}

// MarkDLQRequeued moves the DLQ item back into PENDING for retry
// (used by admin retry endpoint to prevent concurrent duplicate retries)
func (s *Store) MarkDLQRequeued(id int64) error {
	_, err := s.db.Exec(
		context.Background(),
		`UPDATE delivery_dlq
		   SET status = $1,
		       available_at = NOW(),
		       resolved_at = NULL
		 WHERE id = $2`,
		DLQStatusRequeued,
		id,
	)
	return err
}

// GetDueDLQ returns DLQ entries that are eligible for retry (available_at <= now)
func (s *Store) GetDueDLQ(merchantID int64, limit int) ([]DeliveryDLQ, error) {
	now := time.Now().UTC()
	rows, err := s.db.Query(
		context.Background(),
		`SELECT id, transaction_id, merchant_id, webhook_url, attempt_count, last_error, last_response_code, status, available_at, created_at, resolved_at
		 FROM delivery_dlq
		 WHERE merchant_id = $1
		   AND status IN ($2, $3)
		   AND available_at <= $4
		 ORDER BY available_at ASC, created_at DESC
		 LIMIT $5`,
		merchantID,
		DLQStatusPending,
		DLQStatusRequeued,
		now,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DeliveryDLQ
	for rows.Next() {
		var d DeliveryDLQ
		if err := rows.Scan(
			&d.ID,
			&d.TransactionID,
			&d.MerchantID,
			&d.WebhookURL,
			&d.AttemptCount,
			&d.LastError,
			&d.LastResponseCode,
			&d.Status,
			&d.AvailableAt,
			&d.CreatedAt,
			&d.ResolvedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) ListDLQ(merchantID int64, limit, offset int) ([]DeliveryDLQ, error) {
	rows, err := s.db.Query(
		context.Background(),
		`SELECT id, transaction_id, merchant_id, webhook_url, attempt_count, last_error, last_response_code, status, available_at, created_at, resolved_at
		 FROM delivery_dlq
		 WHERE merchant_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		merchantID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DeliveryDLQ
	for rows.Next() {
		var d DeliveryDLQ
		if err := rows.Scan(
			&d.ID,
			&d.TransactionID,
			&d.MerchantID,
			&d.WebhookURL,
			&d.AttemptCount,
			&d.LastError,
			&d.LastResponseCode,
			&d.Status,
			&d.AvailableAt,
			&d.CreatedAt,
			&d.ResolvedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) GetDLQ(id int64, merchantID int64) (*DeliveryDLQ, error) {
	d := &DeliveryDLQ{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, transaction_id, merchant_id, webhook_url, attempt_count, last_error, last_response_code, status, available_at, created_at, resolved_at
		 FROM delivery_dlq
		 WHERE id = $1 AND merchant_id = $2`,
		id,
		merchantID,
	).Scan(
		&d.ID,
		&d.TransactionID,
		&d.MerchantID,
		&d.WebhookURL,
		&d.AttemptCount,
		&d.LastError,
		&d.LastResponseCode,
		&d.Status,
		&d.AvailableAt,
		&d.CreatedAt,
		&d.ResolvedAt,
	)
	return d, err
}

func (s *Store) GetTransactionInternal(transactionID int64) (*Transaction, error) {
	tx := &Transaction{}
	err := s.db.QueryRow(
		context.Background(),
		`SELECT id, merchant_id, provider_tx_id, external_id, amount, currency, status, payer_msisdn, raw_payload, received_at, created_at
		 FROM transactions
		 WHERE id = $1`,
		transactionID,
	).Scan(
		&tx.ID,
		&tx.MerchantID,
		&tx.ProviderTxID,
		&tx.ExternalID,
		&tx.Amount,
		&tx.Currency,
		&tx.Status,
		&tx.PayerMSISDN,
		&tx.RawPayload,
		&tx.ReceivedAt,
		&tx.CreatedAt,
	)
	return tx, err
}

func (s *Store) GetMetrics(merchantID int64) (*Metrics, error) {
	metrics := &Metrics{}

	err := s.db.QueryRow(context.Background(),
		`SELECT
            COUNT(*)                                          AS total,
            COUNT(*) FILTER (WHERE status = 'SUCCESSFUL')    AS successful,
            COUNT(*) FILTER (WHERE status = 'FAILED')        AS failed
         FROM transactions
         WHERE merchant_id = $1`,
		merchantID,
	).Scan(
		&metrics.TransactionsTotal,
		&metrics.TransactionsSuccessful,
		&metrics.TransactionsFailed,
	)
	if err != nil {
		return nil, err
	}

	if metrics.TransactionsTotal > 0 {
		rate := float64(metrics.TransactionsSuccessful) / float64(metrics.TransactionsTotal) * 100
		metrics.DeliverySuccessRate = fmt.Sprintf("%.1f%%", rate)
	} else {
		metrics.DeliverySuccessRate = "N/A"
	}

	return metrics, nil
}
