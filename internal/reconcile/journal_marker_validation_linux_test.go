//go:build linux

package reconcile

import (
	"testing"
	"os"
	"errors"
	"context"
	"path/filepath"

	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

func TestMarkerValidationPending(t *testing.T) {
	if journalMarkerContents == "" {
		t.Fatal("empty marker version")
	}
}

func TestMarkerRejectsDeletedJournal(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil { t.Fatal(err) }
	path := sourceLedgerPath(store.Path)
	if err := writeSourceLedger(path, diskLedger{Version: 1, Records: map[string]record{}}); err != nil { t.Fatal(err) }
	if err := os.Remove(path); err != nil { t.Fatal(err) }
	source := &DiskSource{Directory: d}
	if _, err := source.Count("tenant-one", "subject-one"); !errors.Is(err, ErrJournalMissing) {
		t.Fatalf("missing journal was reset: %v", err)
	}
	if _, err := source.Inspect(context.Background(), "tenant-one", "subject-one", "op-old"); !errors.Is(err, ErrJournalMissing) {
		t.Fatalf("missing journal was inspected as empty: %v", err)
	}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-old", 1)); !errors.Is(err, ErrJournalMissing) {
		t.Fatalf("replay accepted after journal loss: %v", err)
	}
}
