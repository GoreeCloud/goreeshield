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
		Version:       Version,
		Issuer:        "fixture-owner",
		KeyID:         "fixture-key",
		ScopeDigest:   headScope(),
		Sequence:      sequence,
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

func TestDiskHeadRefusesConflictingRevisionAndOwner(t *testing.T) {
	store := DiskHeadStore{Root: filepath.Join(t.TempDir(), "source-owned")}
	if err := store.Initialize(headStatement(1)); err != nil {
		t.Fatal(err)
	}
	if err := store.Advance(headStatement(2)); err != nil {
		t.Fatal(err)
	}
	conflict := headStatement(2)
	conflict.JournalDigest = sha256.Sum256([]byte("changed bytes"))
	if err := store.Advance(conflict); !errors.Is(err, ErrHeadConflict) {
		t.Fatalf("same sequence conflict accepted: %v", err)
	}
	wrongOwner := headStatement(3)
	wrongOwner.Issuer = "other-source"
	if err := store.Advance(wrongOwner); !errors.Is(err, ErrUntrusted) {
		t.Fatalf("source substitution accepted: %v", err)
	}
	last, err := store.Read(headScope())
	if err != nil || last != headStatement(2) {
		t.Fatalf("rejected change modified head: %+v %v", last, err)
	}
}

func TestDiskHeadRejectsMissingAndDamagedHistory(t *testing.T) {
	store := DiskHeadStore{Root: filepath.Join(t.TempDir(), "source-owned")}
	if _, err := store.Read(headScope()); !errors.Is(err, ErrHeadAbsent) {
		t.Fatalf("new source unexpectedly has history: %v", err)
	}
	if err := store.Advance(headStatement(2)); !errors.Is(err, ErrHeadAbsent) {
		t.Fatalf("ordinary update initialized absent source: %v", err)
	}
	if err := store.Initialize(headStatement(1)); err != nil {
		t.Fatal(err)
	}
	path := store.path(headScope())
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(headScope()); !errors.Is(err, ErrHeadCorrupt) {
		t.Fatalf("missing initialized history accepted: %v", err)
	}
	if err := store.Initialize(headStatement(1)); !errors.Is(err, ErrHeadCorrupt) {
		t.Fatalf("bootstrap reset initialized history: %v", err)
	}
	if err := store.Advance(headStatement(2)); !errors.Is(err, ErrHeadCorrupt) {
		t.Fatalf("update reset missing history: %v", err)
	}
}

func TestDiskHeadRejectsCorruptAndReadableState(t *testing.T) {
	store := DiskHeadStore{Root: filepath.Join(t.TempDir(), "source-owned")}
	if err := store.Initialize(headStatement(1)); err != nil {
		t.Fatal(err)
	}
	path := store.path(headScope())
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(headScope()); !errors.Is(err, ErrHeadCorrupt) {
		t.Fatalf("overly readable head accepted: %v", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Advance(headStatement(2)); !errors.Is(err, ErrHeadCorrupt) {
		t.Fatalf("corrupt head was replaced: %v", err)
	}
}

func TestDiskHeadInterruptedFirstInitializationFailsClosed(t *testing.T) {
	store := DiskHeadStore{Root: filepath.Join(t.TempDir(), "source-owned")}
	if err := store.privateRoot(); err != nil {
		t.Fatal(err)
	}
	if err := initializeHeadMarker(store.path(headScope())); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(headStatement(1)); !errors.Is(err, ErrHeadCorrupt) {
		t.Fatalf("unfinished bootstrap silently reset: %v", err)
	}
}
