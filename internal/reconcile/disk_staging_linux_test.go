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

func TestStaleStagingFileDoesNotOverrideCommittedSource(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil { t.Fatal(err) }
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil { t.Fatal(err) }
	source := &DiskSource{Directory: d}
	req := fixtureRequest("op-committed", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	tmp, err := os.CreateTemp(d.Root, ".journal-*.tmp")
	if err != nil { t.Fatal(err) }
	if _, err := tmp.WriteString("not a committed record"); err != nil { t.Fatal(err) }
	if err := tmp.Close(); err != nil { t.Fatal(err) }
	reloaded := &DiskSource{Directory: d}
	obs, err := reloaded.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID)
	if err != nil || !obs.Found || obs.Status != "applied" {
		t.Fatalf("staged data overrode committed source: %+v %v", obs, err)
	}
	if err := reloaded.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	count, err := reloaded.Count("tenant-one", "subject-one")
	if err != nil || count != 1 { t.Fatalf("replayed source counter %d: %v", count, err) }
}
