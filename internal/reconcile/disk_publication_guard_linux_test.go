//go:build linux

package reconcile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// A writer called directly must not bypass the missing-history check that
// already protects normal DiskSource calls.
func TestDirectWriteRejectsMissingInitializedJournal(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	state := diskLedger{Version: 1, Records: map[string]record{}}
	if err := writeSourceLedger(path, state); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := writeSourceLedger(path, state); !errors.Is(err, ErrJournalMissing) {
		t.Fatalf("direct journal write erased missing-history evidence: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing journal was unexpectedly recreated: %v", err)
	}
}

func TestDirectWritePreservesCorruptedInitializedJournal(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	state := diskLedger{Version: 1, Records: map[string]record{}}
	if err := writeSourceLedger(path, state); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte("{not-json")
	if err := os.WriteFile(path, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeSourceLedger(path, state); err == nil {
		t.Fatal("direct writer overwrote corrupted authoritative history")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(corrupt) {
		t.Fatalf("damaged source record was modified: %q %v", data, err)
	}
}

func TestFirstPublicationFailureCannotBeRetriedAsNew(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	state := diskLedger{Version: 1, Records: map[string]record{}}
	stop := errors.New("synthetic pause before journal publication")
	err = writeSourceLedgerWithCheckpoint(path, state, func(stage string) error {
		if stage == "before_rename" {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) {
		t.Fatalf("first publication did not reach checkpoint: %v", err)
	}
	if err := writeSourceLedger(path, state); !errors.Is(err, ErrJournalMissing) {
		t.Fatalf("direct retry bypassed first-publication recovery gate: %v", err)
	}
}
