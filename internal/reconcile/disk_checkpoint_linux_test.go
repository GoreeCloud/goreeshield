//go:build linux

package reconcile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

var errSyntheticCheckpoint = errors.New("synthetic write checkpoint")

func TestJournalCheckpointBeforeRenamePreservesOldRecord(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	previous := diskLedger{Version: 1, Records: make(map[string]record)}
	if err := writeSourceLedger(path, previous); err != nil {
		t.Fatal(err)
	}

	req := fixtureRequest("op-checkpoint", 1)
	next := diskLedger{Version: 1, Count: 1, Records: map[string]record{
		req.OperationID: {Request: req, Status: "applied"},
	}}
	seen := ""
	err = writeSourceLedgerWithCheckpoint(path, next, func(phase string) error {
		seen = phase
		return errSyntheticCheckpoint
	})
	if seen != "before_rename" || !errors.Is(err, errSyntheticCheckpoint) || errors.Is(err, ErrCommitUncertain) {
		t.Fatalf("unexpected pre-rename result: phase %q, error %v", seen, err)
	}
	got, err := readSourceLedger(path)
	if err != nil || got.Count != 0 || len(got.Records) != 0 {
		t.Fatalf("uncommitted candidate replaced prior journal: %+v %v", got, err)
	}
	temps, err := filepath.Glob(filepath.Join(d.Root, ".journal-*.tmp"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("writer leaked temporary journal: %v %v", temps, err)
	}
}

func TestJournalCheckpointAfterRenameIsUncertainAndRecoverable(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	req := fixtureRequest("op-checkpoint", 1)
	ledger := diskLedger{Version: 1, Count: 1, Records: map[string]record{
		req.OperationID: {Request: req, Status: "applied"},
	}}
	err = writeSourceLedgerWithCheckpoint(path, ledger, func(phase string) error {
		if phase == "after_rename" {
			return errSyntheticCheckpoint
		}
		return nil
	})
	if !errors.Is(err, ErrCommitUncertain) {
		t.Fatalf("post-rename error must have uncertain classification: %v", err)
	}
	fresh := &DiskSource{Directory: d}
	observed, err := fresh.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID)
	if err != nil || !observed.Found || observed.Status != "applied" || observed.Request != req {
		t.Fatalf("renamed journal could not be read back: %+v %v", observed, err)
	}
	if err := fresh.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	count, err := fresh.Count("tenant-one", "subject-one")
	if err != nil || count != 1 {
		t.Fatalf("readback retry repeated synthetic source record: %d %v", count, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("source record not private after rename: %v %v", info, err)
	}
}
