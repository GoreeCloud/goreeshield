//go:build linux

package reconcile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// TestInterruptedJournalHelper is a separate test process used solely for
// local, synthetic recovery experiments.
func TestInterruptedJournalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_JOURNAL_INTERRUPTION_HELPER") != "1" {
		return
	}
	root := os.Getenv("JOURNAL_INTERRUPTION_ROOT")
	if root == "" {
		t.Fatal("missing fixture root")
	}
	source := &DiskSource{Directory: scoped.Directory{Root: root}}
	req := fixtureRequest("op-interrupted", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stdout.WriteString("SAVED\n"); err != nil {
		t.Fatal(err)
	}
	<-make(chan struct{})
}

func TestInterruptedJournalOrphanTemporaryFileIsNotTrusted(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(filepath.Dir(store.Path), ".journal-partial.tmp")
	if err := os.WriteFile(partial, []byte("{\"version\":1,\"count\":"), 0600); err != nil {
		t.Fatal(err)
	}
	source := &DiskSource{Directory: d}
	count, err := source.Count("tenant-one", "subject-one")
	if err != nil || count != 0 {
		t.Fatalf("orphan temp affected the journal: %d %v", count, err)
	}
}
