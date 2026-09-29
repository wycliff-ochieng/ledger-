package audit

import (
	"bytes"
	"testing"
)

func TestHashLeaf(t *testing.T) {
	data := []byte("leaf data")
	hash := HashLeaf(data)
	if len(hash) != 32 { // SHA-256 length is 32 bytes
		t.Errorf("expected hash length 32, got %d", len(hash))
	}
}

func TestHashInterior(t *testing.T) {
	left := []byte("left_node_hash")
	right := []byte("right_node_hash")
	hash := HashInterior(left, right)
	if len(hash) != 32 {
		t.Errorf("expected hash length 32, got %d", len(hash))
	}
}

func TestBuildMerkleTree_ZeroLeaves(t *testing.T) {
	_, err := BuildMerkleTree(nil)
	if err == nil {
		t.Error("expected error when building Merkle Tree with zero leaves")
	}
}

func TestBuildMerkleTree_SingleLeaf(t *testing.T) {
	leaves := [][]byte{[]byte("single_leaf")}
	tree, err := BuildMerkleTree(leaves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree.Root == nil {
		t.Fatal("expected a non-nil root")
	}
	if !bytes.Equal(tree.Root.Hash, leaves[0]) {
		t.Errorf("expected root hash to equal single leaf hash")
	}
	if len(tree.Leafs) != 1 {
		t.Errorf("expected exactly 1 leaf node, got %d", len(tree.Leafs))
	}
}

func TestBuildMerkleTree_EvenLeaves(t *testing.T) {
	leaves := [][]byte{
		[]byte("leaf1"),
		[]byte("leaf2"),
		[]byte("leaf3"),
		[]byte("leaf4"),
	}
	tree, err := BuildMerkleTree(leaves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree.Root == nil {
		t.Fatal("expected non-nil root")
	}
	if len(tree.Leafs) != 4 {
		t.Errorf("expected 4 leaf nodes")
	}
}

func TestBuildMerkleTree_OddLeaves(t *testing.T) {
	leaves := [][]byte{
		[]byte("leaf1"),
		[]byte("leaf2"),
		[]byte("leaf3"),
	}
	tree, err := BuildMerkleTree(leaves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree.Root == nil {
		t.Fatal("expected non-nil root")
	}
	if len(tree.Leafs) != 3 {
		t.Errorf("expected 3 leaf nodes")
	}
}
