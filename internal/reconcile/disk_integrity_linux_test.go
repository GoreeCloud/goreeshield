//go:build linux

package reconcile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

func TestJournalHardlinkMustBeRejected(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	if err := os.WriteFile(path, []byte("{\"version\":1,\"count\":0,\"records\":{}}"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "journal-hardlink")
	if err := os.Link(path, link); err != nil {
		t.Fatal(err)
	}
	source := &DiskSource{Directory: d}
	if _, err := source.Count("tenant-one", "subject-one"); err == nil {
		t.Fatal("multiply-linked journal was trusted")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Count("tenant-one", "subject-one"); err != nil {
		t.Fatalf("journal should be readable after hardlink removed: %v", err)
	}
	lock := path + ".lock"
	lockLink := filepath.Join(t.TempDir(), "lock-hardlink")
	if err := os.Link(lock, lockLink); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Inspect(context.Background(), "tenant-one", "subject-one", "op-unknown"); err == nil {
		t.Fatal("multiply-linked lock file was trusted")
	}
}

func TestJournalRejectsCountAndStatusTampering(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	source := &DiskSource{Directory: d}
	for _, sample := range []string{
		"{\"version\":1,\"count\":1,\"records\":{}}",
		"{\"version\":9,\"count\":0,\"records\":{}}",
		"{\"version\":1,\"count\":-1,\"records\":{}}",
		"{\"version\":1,\"count\":0,\"records\":{}} {}",
		"{\"version\":1,\"count\":0,\"records\":{},\"unknown\":true}",
	} {
		if err := os.WriteFile(path, []byte(sample), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := source.Count("tenant-one", "subject-one"); err == nil {
			t.Fatalf("unsafe source journal unexpectedly accepted: %s", sample)
		}
	}
}
