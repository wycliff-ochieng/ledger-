package repository

import (
	"Wycliff-Ochieng/audit"
	"Wycliff-Ochieng/internal/models"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ LedgerRepository = (*PostgresRepo)(nil)

func (r *PostgresRepo) CreateAccount(ctx context.Context, account *models.Account) error {
	query := `
		INSERT INTO accounts (id, ledger_id, name, type, normal_balance, currency, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		account.AccountID,
		account.LedgerID,
		account.Name,
		account.Type,
		account.NormalBalance,
		account.Currency,
		account.CreatedAt,
	)
	return err
}

func (r *PostgresRepo) GetAccount(ctx context.Context, accountID uuid.UUID) (*models.Account, error) {
	query := `
		SELECT id, ledger_id, name, type, normal_balance, currency, created_at
		FROM accounts
		WHERE id = $1
	`
	var acc models.Account
	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&acc.AccountID,
		&acc.LedgerID,
		&acc.Name,
		&acc.Type,
		&acc.NormalBalance,
		&acc.Currency,
		&acc.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &acc, nil
}

func (r *PostgresRepo) GetAccountByID(ctx context.Context, accountIDs []uuid.UUID) map[uuid.UUID]string {
	// Not fully implemented yet, maybe meant to return currency or name?
	// The interface signature says map[uuid.UUID]string, so we'll just mock it or implement a basic fetch.
	return nil
}

func (r *PostgresRepo) SaveTransactions(ctx context.Context, tx *models.Transaction, leafHash []byte) error {
	pgxTx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer pgxTx.Rollback(ctx)

	// 1. Insert Transaction
	txQuery := `
		INSERT INTO transactions (id, ledger_id, idempotency_key, description, posted_at, tx_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = pgxTx.Exec(ctx, txQuery,
		tx.TransactionID,
		tx.LedgerID,
		tx.IdempotencyKey.String(),
		tx.Description,
		tx.PostedAt,
		leafHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 = unique_violation
			return ErrDuplicateIdempotencyKey
		}
		return err
	}

	// 2. Insert Entries
	entryQuery := `
		INSERT INTO ledger_entries (id, transaction_id, account_id, amount, direction, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	// Batch insert for performance
	batch := &pgx.Batch{}
	for _, entry := range tx.Entries {
		batch.Queue(entryQuery,
			entry.EntryID,
			entry.TransactionID,
			entry.AccountID,
			entry.Amount,
			entry.Direction,
			entry.CreatedAt,
		)
	}

	batchResults := pgxTx.SendBatch(ctx, batch)
	for i := 0; i < len(tx.Entries); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			batchResults.Close()
			return err
		}
	}
	batchResults.Close()

	// 3. Insert Audit Outbox
	auditQuery := `
		INSERT INTO audit_outbox (transaction_id, leaf_hash, status, created_at)
		VALUES ($1, $2, 'PENDING', $3)
	`
	_, err = pgxTx.Exec(ctx, auditQuery, tx.TransactionID, leafHash, tx.PostedAt)
	if err != nil {
		return err
	}

	// Commit everything atomically
	return pgxTx.Commit(ctx)
}

func (r *PostgresRepo) FetchPendingAuditLogs(ctx context.Context, batchSize int) ([]*models.AuditLogEntry, error) {
	// Not fully implemented for phase 2 initial requirement
	return nil, nil
}

func (r *PostgresRepo) FinalizeEpoch(ctx context.Context, epoch *audit.MerkleEpoch, txIDs []uuid.UUID) error {
	// Not fully implemented for phase 2 initial requirement
	return nil
}

func (r *PostgresRepo) GetEpoch(ctx context.Context, epochID uuid.UUID) (*audit.MerkleEpoch, []*models.Transaction, error) {
	// Not fully implemented for phase 2 initial requirement
	return nil, nil, nil
}
