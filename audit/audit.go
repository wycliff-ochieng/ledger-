package audit

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/google/uuid"
)

//Merkle Tree - Cryprographic data structure that organizes large blocks of data into a tree of hashes
//condensing everything into a single top-level value called "Merkle Root"
// How it works :
// - Leaf Nodes - Every individual piece of transaction at the bottom is hashed
// - Branch Nodes - Adjacent pairs of hashes are combined,concatenated and hashed again to  create parent nodes
// - Merkle Root  - The process repeats upward until a single final hash remains at the top, representing the entire dataset
// - Tamper Evidence - if any underlying data changes the resulting hash chnages all the way up to the root, immediately exposing modifications / alterations

// Importance , Why use Merkle Trees??
// - Efficient Verfication : Users can prove specific transaction or piece of data belongs in the set using a small "Merkle Proof" instead of downloading the whole dataset
// - Blockchain use: securing blocks and clients are able to verify trasactions efficiently

//Epoch based Merkle Pipeline: The audit Layer must??

// 1. - zero impact on write throughput : posting a transaction must complete in < 25 ms.
// chaining a global tree synchronously in-band introduces contention bottlencks and makes the write dependent on complex hashing operations
// 2. - Deterministic Tamper-Evidence : any modification to the transactions payload  / historical sequence results  in mathematically invalid proof

/*

[ POST /v1/transactions ]
         │
         ▼ (Atomic DB Transaction)
┌────────────────────────────────────────────────────────────┐
│ 1. Verify Balances & Insert Entries                        │
│ 2. Compute Deterministic Leaf Hash: H(tx)                  │
│ 3. INSERT INTO transactions (..., tx_hash)                 │
│ 4. INSERT INTO audit_outbox (tx_id, tx_hash, created_at)   │
└────────────────────────────────────────────────────────────┘
         │
         ▼ (Async Polling / CDC)
┌────────────────────────────────────────────────────────────┐
│                  Audit Epoch Worker                        │
│                                                            │
│  - Select unbatched events from audit_outbox (e.g. 1,000)  │
│  - Sort deterministically by (posted_at ASC, id ASC)       │
│  - Construct Binary Merkle Tree                            │
│  - Persist Root Hash & Merkle Nodes to merkle_epochs       │
│  - Mark outbox items as BATCHED                            │
│  - (Optional) Publish Root to External Append-Only Store   │
└────────────────────────────────────────────────────────────┘


*/

type AuditService interface {
	ProcessPendingEpoch(ctx context.Context, batchSize int64) (*MerkleEpoch, error)
	VerifyEpoch(ctx context.Context, epochID uuid.UUID) (VerficationResult, error)
}

type MerkleNode struct {
	Hash  []byte
	Left  *MerkleNode
	Right *MerkleNode
}

type MerkleTree struct {
	Root  *MerkleNode
	Leafs []*MerkleNode
}

type MerkleEpoch struct {
	EpochID          uuid.UUID
	EpochNumber      int64
	RootHash         []byte
	PreviousEpoch    []byte
	TransactionCount int64
	CreatedAT        time.Time
}

type VerficationResult struct {
	EpochID         uuid.UUID
	ExpectedRoot    string
	RecomputedRooot string
	IsValid         bool
	TamperedTxID    uuid.UUID
}

// produces deterministic SHA-256 digest of raw leaf data
func HashLeaf(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// combines two child node digests: SHA-256(left || right)
func HashInterior(left, right []byte) []byte {
	h := sha256.New()
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// construct a balanced binary tree from a slice of leaf hashes
func BuildMerkleTree(leafHashes [][]byte) (*MerkleTree, error) {
	if len(leafHashes) == 0 {
		return nil, errors.New("cannot build Merkle Tree with zero leaf hashes")
	}

	var nodes []*MerkleNode
	for _, h := range leafHashes {
		nodes = append(nodes, &MerkleNode{Hash: h})
	}

	leafNodes := make([]*MerkleNode, len(nodes))
	copy(leafNodes, nodes)

	//build levels upward until we reach the root
	for len(nodes) > 1 {
		//if uneven , duplicate last node to preserve binary structure
		if len(nodes)%2 != 0 {
			nodes = append(nodes, &MerkleNode{Hash: nodes[len(nodes)-1].Hash})
		}

		var parentNodes []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			combined := HashInterior(nodes[i].Hash, nodes[i+1].Hash)
			parent := &MerkleNode{
				Hash:  combined,
				Left:  nodes[i],
				Right: nodes[i+1],
			}

			parentNodes = append(parentNodes, parent)
		}
		nodes = parentNodes
	}
	return &MerkleTree{
		Root:  nodes[0],
		Leafs: leafNodes,
	}, nil

}
