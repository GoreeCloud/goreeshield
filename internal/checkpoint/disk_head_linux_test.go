//go:build linux

package checkpoint

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func headScope() [32]byte {
	return sha256.Sum256([]byte("synthetic source scope"))
}

func headStatement(sequence uint64) Statement {
	return Statement{
		Version: Version,
		Issuer: "fixture-owner",
		KeyID: "fixture-key",
		ScopeDigest: headScope(),
		Sequence: sequence,
		JournalDigest: sha256.Sum256([]byte{byte(sequence)}),
	}
}

func TestDiskHeadPersistsAndEnforcesSequence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "source-owned")
	store := DiskHeadStore{Root: root}
	first := headStatement(1)
	if err := store.Initialize(first); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(first); !errors.Is(err, ErrHeadConflict) {
		t.Fatalf("duplicate bootstrap accepted: %v", err)
	}
	restarted := DiskHeadStore{Root: root}
	read, err := restarted.Read(first.ScopeDigest)
	if err != nil || read != first {
		t.Fatalf("new reader did not recover first head: %+v %v", read, err)
	}
	next := headStatement(2)
	if err := restarted.Advance(next); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Advance(next); err != nil {
		t.Fatalf("idempotent retry failed: %v", err)
	}
	if err := restarted.Advance(first); !errors.Is(err, ErrHeadRegression) {
		t.Fatalf("sequence regression accepted: %v", err)
	}
	if err := restarted.Advance(headStatement(4)); !errors.Is(err, ErrHeadGap) {
		t.Fatalf("sequence gap accepted: %v", err)
	}
}
