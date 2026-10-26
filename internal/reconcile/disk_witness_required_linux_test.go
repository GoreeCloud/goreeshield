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

func TestWitnessRequiredAfterSuccessfulSourcePublication(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	source := &DiskSource{Directory: dir}
	req := fixtureRequest("op-required-witness", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	required, err := journalRequiresWitness(path)
	if err != nil || !required {
		t.Fatalf("successful local source write lacks witness requirement: %v %v", required, err)
	}
	if err := os.Remove(journalWitnessPath(path)); err != nil {
		t.Fatal(err)
	}
	reopened := &DiskSource{Directory: dir}
	if _, err := reopened.Count("tenant-one", "subject-one"); !errors.Is(err, ErrWitnessMissing) {
		t.Fatalf("deleted mandatory witness was accepted: %v", err)
	}
	if _, err := reopened.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID); !errors.Is(err, ErrWitnessMissing) {
		t.Fatalf("missing-witness readback falsely succeeded: %v", err)
	}
	if err := reopened.Apply(context.Background(), "tenant-one", "subject-one", req); !errors.Is(err, ErrWitnessMissing) {
		t.Fatalf("missing witness permitted replay: %v", err)
	}
}

func TestLegacyWitnessMigrationBecomesMandatory(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	if err := os.WriteFile(path, []byte("{\"version\":1,\"count\":0,\"records\":{}}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalMarkerPath(path), []byte(journalMarkerContents), 0600); err != nil {
		t.Fatal(err)
	}
	required, err := journalRequiresWitness(path)
	if err != nil || required {
		t.Fatalf("legacy marker unexpectedly requires witness: %v %v", required, err)
	}
	source := &DiskSource{Directory: dir}
	if count, err := source.Count("tenant-one", "subject-one"); err != nil || count != 0 {
		t.Fatalf("legacy source failed to open: %d %v", count, err)
	}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-legacy-promote", 1)); err != nil {
		t.Fatal(err)
	}
	required, err = journalRequiresWitness(path)
	if err != nil || !required {
		t.Fatalf("legacy scope was not promoted after successful write: %v %v", required, err)
	}
	if err := os.Remove(journalWitnessPath(path)); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Count("tenant-one", "subject-one"); !errors.Is(err, ErrWitnessMissing) {
		t.Fatalf("promoted scope accepted missing witness: %v", err)
	}
}

func TestDirectWriterCannotBypassRequiredWitness(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	journal := diskLedger{Version: 1, Records: map[string]record{}}
	if err := writeSourceLedger(path, journal); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(journalWitnessPath(path)); err != nil {
		t.Fatal(err)
	}
	if err := writeSourceLedger(path, journal); !errors.Is(err, ErrWitnessMissing) {
		t.Fatalf("direct writer bypassed mandatory witness check: %v", err)
	}
	if _, err := os.Stat(journalWitnessPath(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("direct writer silently recreated witness: %v", err)
	}
}
