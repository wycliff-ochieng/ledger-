package repository

import (
	"Wycliff-Ochieng/audit"
	"Wycliff-Ochieng/internal/models"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDuplicateIdempotencyKey = errors.New("idempotency key already processed")
	ErrAccountNotFound         = errors.New("one or more account do not exist")
	ErrInsufficientFunds       = errors.New("insufficient funds for debit operations")
)

/*
 Epoch - discrete batch of operations
 in our case Epoch -> bounded batch of transactions sealed together into a single cryptographic unit

 -instead of recalculating MerkleTree after every individual transaction which will destroy database write throughput
 we divide continuous time or transactions into epochs



*/

type LedgerRepository interface {
	CreateAccount(Ctx context.Context, account *models.Account) error
	GetAccount(ctx context.Context, accountID uuid.UUID) (*models.Account, error)
	GetAccountByID(ctx context.Context, accountIDs []uuid.UUID) map[uuid.UUID]string

	SaveTransactions(ctx context.Context, tx *models.Transaction, leafHash []byte) error
	//FetchPendingAuditLogs(ctx context.Context, tx *models.Transaction)
	FetchPendingAuditLogs(ctx context.Context, batchSize int) ([]*models.AuditLogEntry, error)
	FinalizeEpoch(ctx context.Context, epoch *audit.MerkleEpoch, txIDs []uuid.UUID) error
	GetEpoch(ctx context.Context, epochID uuid.UUID) (*audit.MerkleEpoch, []*models.Transaction, error)
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{
		pool: pool,
	}
}
