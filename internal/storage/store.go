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

// TransactionExists satisfies the auth.SeenChecker interface.
// Uses EXISTS so Postgres stops at the first match — no full scan.
func (s *Store) TransactionExists(providerTxID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		context.Background(),
		`SELECT EXISTS(SELECT 1 FROM transaction WHERE provider_tx_id = $1)`,
		providerTxID,
	).Scan(&exists)
	return exists, err
}
