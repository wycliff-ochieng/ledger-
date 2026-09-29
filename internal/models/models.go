package models

import (
	"Wycliff-Ochieng/audit"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

type AccountType string

var (
	ErrInsuffucientEntries = errors.New("insufficient entries")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidDirection    = errors.New("invalid direction flow")
	ErrUnbalancedLedger    = errors.New("unbalanced ledger")
)

const (
	AccountTypeAsset     AccountType = "ASSETS"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeExpense   AccountType = "EXPENSE"
	AccountTypeEquity    AccountType = "EQUITY"
	AccountTypeRevenue   AccountType = "REVENUE"
)

type Direction string

const (
	DirectionDebit  Direction = "DEBIT"
	DirectionCredit Direction = "CREDIT"
)

type Account struct {
	AccountID     uuid.UUID `json:"AccountID"`
	LedgerID      uuid.UUID `json:"ledgerID"`
	Name          string
	Type          AccountType `json:"accountType"`
	Currency      string
	NormalBalance Direction `json:"normalBalance"`
	CreatedAt     time.Time
}

type Transaction struct {
	TransactionID  uuid.UUID `json:"transaction_id"`
	LedgerID       uuid.UUID `json:"ledger_id"`
	IdempotencyKey uuid.UUID `json:"idempotency_key"`
	Description    string    `json:"description"`
	PostedAt       time.Time `json:"posted_at"`
	Entries        []Entry   `json:"entries"`
}

// strictly append-only no delete or update
type Entry struct {
	EntryID       uuid.UUID
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Amount        uint64
	Direction     Direction
	CreatedAt     time.Time
}

type AuditStatus string

type AuditLogEntry struct {
	TransactionID uuid.UUID
	LeafHash      []byte
	EpochID       uuid.UUID
	Status        AuditStatus
	CreatedAt     time.Time
}

func (t *Transaction) Validate() error {
	//check entries - must be atleast 2 entries

	if len(t.Entries) < 2 {
		return ErrInsuffucientEntries
	}

	var totalDebit uint64
	var totalCredit uint64
	//for every entry amount > 0

	for i, entry := range t.Entries {
		if entry.Amount == 0 {
			return ErrInvalidAmount
		}

		switch entry.Direction {
		case DirectionDebit:
			totalDebit += entry.Amount
		case DirectionCredit:
			totalCredit += entry.Amount
		default:
			return fmt.Errorf("%w at index %d: %v", ErrInvalidDirection, i, entry.Direction)
		}
	}
	// sum(DEBITS) == sum(CREDITS)

	if totalDebit != totalCredit {
		return fmt.Errorf("%w: total debits(%d) != total credits(%d)", ErrUnbalancedLedger, totalDebit, totalCredit)
	}
	return nil
}

// compute SHA-256 hash across all fields and entries
func (t *Transaction) ComputeLeafHash() []byte {
	//sort entries dynamically by accountID and direction

	sortedEntries := make([]Entry, len(t.Entries))
	copy(sortedEntries, t.Entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		if sortedEntries[i].AccountID == sortedEntries[j].AccountID {
			return sortedEntries[i].Direction < sortedEntries[j].Direction
		}
		return sortedEntries[i].AccountID.String() < sortedEntries[j].AccountID.String()
	})

	//digest the entries
	entryHasher := sha256.New()

	for _, e := range sortedEntries {
		entryHasher.Write(e.AccountID[:])
		amountBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(amountBytes, uint64(e.Amount))
		entryHasher.Write(amountBytes)
		entryHasher.Write([]byte(e.Direction))
	}

	entriesDigest := entryHasher.Sum(nil)

	//digest the transaction wrapper
	txHasher := sha256.New()
	txHasher.Write(t.TransactionID[:])
	txHasher.Write(t.LedgerID[:])
	txHasher.Write([]byte(t.PostedAt.UTC().Format("2006-01-02T15:04:05.999999999Z")))
	txHasher.Write(t.IdempotencyKey[:])
	txHasher.Write(entriesDigest)

	return txHasher.Sum(nil)
}

func VerifyEpochIntegrity(expectedEpoch []byte, txs []*Transaction) (bool, error) {
	if len(txs) == 0 {
		return false, fmt.Errorf("empty transaction set provided for verfication")
	}

	computedLeafs := make([][]byte, len(txs))
	for i, tx := range txs {
		computedLeafs[i] = tx.ComputeLeafHash()
	}

	tree, err := audit.BuildMerkleTree(computedLeafs)
	if err != nil {
		return false, fmt.Errorf("failed to reconstruct tree : %w", err)
	}

	//compare reconstructed root vs the authoritative stored root
	if string(tree.Root.Hash) != string(expectedEpoch) {
		return false, nil
	}

	return true, nil
}
