//go:build linux

package reconcile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// Corrupt authoritative data must not be silently repaired from an unrelated
// valid-looking staged file. Recovery requires a reviewed manual procedure.
func TestStagingFileCannotRepairCorruptedCommittedJournal(t *testing.T) {
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
	source := &DiskSource{Directory: d}
	req := fixtureRequest("op-before-corruption", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := os.CreateTemp(d.Root, ".journal-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staged.Write(original); err != nil {
		t.Fatal(err)
	}
	if err := staged.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	newSource := &DiskSource{Directory: d}
	if _, err := newSource.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID); err == nil {
		t.Fatal("damaged committed file was silently repaired from staging")
	}
	if err := newSource.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-new", 1)); err == nil {
		t.Fatal("new operation was accepted against damaged authoritative journal")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "{" {
		t.Fatalf("damaged authoritative journal was unexpectedly replaced: %q %v", data, err)
	}
}
