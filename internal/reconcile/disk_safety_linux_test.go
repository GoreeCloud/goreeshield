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

func TestDiskSourceDenialIsPersistentEvenAfterRegrant(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	source := &DiskSource{Directory: d}
	req := fixtureRequest("op-denied", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	restarted := &DiskSource{Directory: d}
	if err := restarted.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	observation, err := restarted.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID)
	if err != nil || !observation.Found || observation.Status != "denied" {
		t.Fatalf("old denied ID revived: %+v %v", observation, err)
	}
	count, err := restarted.Count("tenant-one", "subject-one")
	if err != nil || count != 0 {
		t.Fatalf("denied operation was applied: %d %v", count, err)
	}
}

func TestDiskSourceCorruptedJournalFailsClosed(t *testing.T) {
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
	path := sourceLedgerPath(store.Path)
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-corrupt", 1)); err == nil {
		t.Fatal("malformed journal unexpectedly accepted")
	}
	if _, err := source.Inspect(context.Background(), "tenant-one", "subject-one", "op-corrupt"); err == nil {
		t.Fatal("corrupt journal falsely returned an observation")
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"count":0,"records":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-perms", 1)); err == nil {
		t.Fatal("world-readable valid journal was accepted")
	}
}

func TestDiskSourceRejectsSymlinkedLock(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(t.TempDir(), "real-lock")
	if err := os.WriteFile(real, []byte("do not modify"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, sourceLedgerPath(store.Path)+".lock"); err != nil {
		t.Fatal(err)
	}
	source := &DiskSource{Directory: d}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-symlink", 1)); err == nil {
		t.Fatal("symlinked journal lock accepted")
	}
	original, err := os.ReadFile(real)
	if err != nil || string(original) != "do not modify" {
		t.Fatalf("symlink target unexpectedly modified: %q %v", original, err)
	}
	}
