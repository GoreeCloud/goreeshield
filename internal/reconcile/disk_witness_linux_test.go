//go:build linux

package reconcile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

func TestWitnessDetectsOlderValidJournalCopy(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	source := &DiskSource{Directory: d}
	old := fixtureRequest("op-rollback-old", 1)
	next := fixtureRequest("op-rollback-new", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", old); err != nil {
		t.Fatal(err)
	}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	previous, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", next); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, previous, 0600); err != nil {
		t.Fatal(err)
	}
	restarted := &DiskSource{Directory: d}
	if _, err := restarted.Count("tenant-one", "subject-one"); !errors.Is(err, ErrJournalRollback) {
		t.Fatalf("older but valid JSON source was accepted: %v", err)
	}
	if _, err := restarted.Inspect(context.Background(), "tenant-one", "subject-one", old.OperationID); !errors.Is(err, ErrJournalRollback) {
		t.Fatalf("rollback source inspection was trusted: %v", err)
	}
	if err := restarted.Apply(context.Background(), "tenant-one", "subject-one", next); !errors.Is(err, ErrJournalRollback) {
		t.Fatalf("rollback permitted synthetic replay: %v", err)
	}
}
