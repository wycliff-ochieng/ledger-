package models

import (
	"bytes"
	"testing"
	"time"

	"Wycliff-Ochieng/audit"
	"github.com/google/uuid"
)

func TestTransaction_Validate(t *testing.T) {
	// Test insufficient entries
	tx := &Transaction{
		Entries: []Entry{
			{Amount: 100, Direction: DirectionDebit},
		},
	}
	if err := tx.Validate(); err != ErrInsuffucientEntries {
		t.Errorf("expected %v, got %v", ErrInsuffucientEntries, err)
	}

	// Test zero amount
	tx.Entries = []Entry{
		{Amount: 100, Direction: DirectionDebit},
		{Amount: 0, Direction: DirectionCredit},
	}
	if err := tx.Validate(); err != ErrInvalidAmount {
		t.Errorf("expected %v, got %v", ErrInvalidAmount, err)
	}

	// Test invalid direction
	tx.Entries = []Entry{
		{Amount: 100, Direction: "INVALID_DIRECTION"},
		{Amount: 100, Direction: DirectionCredit},
	}
	if err := tx.Validate(); err == nil {
		t.Error("expected error for invalid direction")
	}

	// Test unbalanced ledger
	tx.Entries = []Entry{
		{Amount: 100, Direction: DirectionDebit},
		{Amount: 200, Direction: DirectionCredit},
	}
	if err := tx.Validate(); err == nil {
		t.Error("expected error for unbalanced ledger")
	}

	// Test valid transaction
	tx.Entries = []Entry{
		{Amount: 100, Direction: DirectionDebit},
		{Amount: 100, Direction: DirectionCredit},
	}
	if err := tx.Validate(); err != nil {
		t.Errorf("unexpected error for valid transaction: %v", err)
	}
}

func TestTransaction_ComputeLeafHash(t *testing.T) {
	tx := &Transaction{
		TransactionID:  uuid.New(),
		LedgerID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		PostedAt:       time.Now().UTC(), // Keep it UTC for clean date formatting
		Entries: []Entry{
			{
				AccountID: uuid.New(),
				Amount:    100,
				Direction: DirectionDebit,
			},
			{
				AccountID: uuid.New(),
				Amount:    100,
				Direction: DirectionCredit,
			},
		},
	}

	hash1 := tx.ComputeLeafHash()
	if len(hash1) != 32 { // SHA-256 hash length is 32 bytes
		t.Errorf("expected hash length 32, got %d", len(hash1))
	}

	// Should be deterministic
	hash2 := tx.ComputeLeafHash()
	if !bytes.Equal(hash1, hash2) {
		t.Error("expected hash to be deterministic")
	}

	// Altering the transaction modifies the hash
	tx.Entries[0].Amount = 200
	hash3 := tx.ComputeLeafHash()
	if bytes.Equal(hash1, hash3) {
		t.Error("expected hash to change when entry amount is altered")
	}
}

func TestVerifyEpochIntegrity(t *testing.T) {
	// Test empty transaction set
	ok, err := VerifyEpochIntegrity(nil, nil)
	if err == nil {
		t.Error("expected error for empty transaction set")
	}
	if ok {
		t.Error("expected ok to be false")
	}

	tx1 := &Transaction{
		TransactionID: uuid.New(),
		PostedAt:      time.Now().UTC(),
		Entries: []Entry{
			{AccountID: uuid.New(), Amount: 100, Direction: DirectionDebit},
			{AccountID: uuid.New(), Amount: 100, Direction: DirectionCredit},
		},
	}
	tx2 := &Transaction{
		TransactionID: uuid.New(),
		PostedAt:      time.Now().UTC(),
		Entries: []Entry{
			{AccountID: uuid.New(), Amount: 200, Direction: DirectionDebit},
			{AccountID: uuid.New(), Amount: 200, Direction: DirectionCredit},
		},
	}

	txs := []*Transaction{tx1, tx2}

	// Calculate manually to get the expected correct root
	computedLeafs := make([][]byte, len(txs))
	for i, tx := range txs {
		computedLeafs[i] = tx.ComputeLeafHash()
	}
	tree, err := audit.BuildMerkleTree(computedLeafs)
	if err != nil {
		t.Fatalf("unexpected error building Merkle tree for test: %v", err)
	}

	correctRoot := tree.Root.Hash

	// Verify valid sequence
	ok, err = VerifyEpochIntegrity(correctRoot, txs)
	if err != nil {
		t.Errorf("unexpected error on VerifyEpochIntegrity: %v", err)
	}
	if !ok {
		t.Error("expected integrity check to pass for correct root")
	}

	// Verify invalid sequence
	invalidRoot := []byte("invalid-root-hash-which-is-just-garbage")
	ok, err = VerifyEpochIntegrity(invalidRoot, txs)
	if err != nil {
		t.Errorf("unexpected error on VerifyEpochIntegrity with invalid root: %v", err)
	}
	if ok {
		t.Error("expected integrity check to fail for incorrect root")
	}
}
