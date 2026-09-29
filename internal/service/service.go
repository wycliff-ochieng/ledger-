package service

import (
	"Wycliff-Ochieng/internal/models"
	"context"

	"github.com/google/uuid"
)

type PostTransactionRequest struct {
	LedgerID       uuid.UUID
	IdempotencyKey uuid.UUID
	Description    string
	Entries        []models.Entry
}

type LedgerService interface {
	CreateAccount(ctx context.Context, name string, accType models.AccountType) (*models.Account, error)
	GetAccountBalance(ctx context.Context, accountID uuid.UUID) (int64, error)
	PostTransaction(ctx context.Context, req PostTransactionRequest) (*models.Transaction, error)
}
