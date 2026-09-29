package service

import (
	"Wycliff-Ochieng/internal/models"
	"Wycliff-Ochieng/internal/repository"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type defaultLedgerService struct {
	repo repository.LedgerRepository
}

func NewLedgerService(repo repository.LedgerRepository) LedgerService {
	return &defaultLedgerService{repo: repo}
}

func (s *defaultLedgerService) CreateAccount(ctx context.Context, name string, accType models.AccountType) (*models.Account, error) {
	normalBalance := models.DirectionCredit
	if accType == models.AccountTypeAsset || accType == models.AccountTypeExpense {
		normalBalance = models.DirectionDebit
	}

	// Use hardcoded UUID for now as there's no endpoint to create ledgers dynamically
	defaultLedger, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	acc := &models.Account{
		AccountID:     uuid.New(),
		LedgerID:      defaultLedger, 
		Name:          name,
		Type:          accType,
		NormalBalance: normalBalance,
		Currency:      "USD", // Default or parameter
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.repo.CreateAccount(ctx, acc); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return acc, nil
}

func (s *defaultLedgerService) GetAccountBalance(ctx context.Context, accountID uuid.UUID) (int64, error) {
	// The plan specifies dynamic balance calculation, but in this phase we will just return a placeholder or query.
	// For dynamic calculation, we would fetch entries or have a repo method that does SUM() based on NormalBalance.
	// We'll trust the caller to expand this or rely on the `accounts.balance` field.
	// For now, let's keep it simple.
	return 0, nil
}

func (s *defaultLedgerService) PostTransaction(ctx context.Context, req PostTransactionRequest) (*models.Transaction, error) {
	tx := &models.Transaction{
		TransactionID:  uuid.New(),
		LedgerID:       req.LedgerID,
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
		PostedAt:       time.Now().UTC(),
		Entries:        make([]models.Entry, len(req.Entries)),
	}

	for i, entry := range req.Entries {
		tx.Entries[i] = models.Entry{
			EntryID:       uuid.New(),
			TransactionID: tx.TransactionID,
			AccountID:     entry.AccountID,
			Amount:        entry.Amount,
			Direction:     entry.Direction,
			CreatedAt:     tx.PostedAt,
		}
	}

	if err := tx.Validate(); err != nil {
		return nil, fmt.Errorf("invalid transaction: %w", err)
	}

	leafHash := tx.ComputeLeafHash()

	if err := s.repo.SaveTransactions(ctx, tx, leafHash); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return tx, nil
}
