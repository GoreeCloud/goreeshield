//go:build linux

package reconcile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// TestDiskJournalSubprocessWorker is invoked only by the cross-process test.
// Every worker has its own Go runtime and DiskSource instance.
func TestDiskJournalSubprocessWorker(t *testing.T) {
	if os.Getenv("GO_WANT_SOURCE_JOURNAL_HELPER") != "1" {
		return
	}
	root := os.Getenv("SOURCE_JOURNAL_ROOT")
	if root == "" {
		t.Fatal("synthetic subprocess fixture requires private root")
	}
	source := &DiskSource{Directory: scoped.Directory{Root: root}}
	req := fixtureRequest("op-across-processes", 1)
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	observed, err := source.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID)
	if err != nil || !observed.Found || observed.Status != "applied" || observed.Request != req {
		t.Fatalf("subprocess source readback mismatch: %+v %v", observed, err)
	}
}

func TestDiskJournalConcurrentOSProcesses(t *testing.T) {
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

	const workers = 4
	var group sync.WaitGroup
	failures := make(chan error, workers)
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			child := exec.Command(os.Args[0], "-test.run=^TestDiskJournalSubprocessWorker$", "-test.v=false")
			child.Env = append(os.Environ(),
				"GO_WANT_SOURCE_JOURNAL_HELPER=1",
				"SOURCE_JOURNAL_ROOT="+d.Root,
			)
			output, err := child.CombinedOutput()
			if err != nil {
				failures <- fmt.Errorf("independent source process failed: %w: %s", err, output)
			}
		}()
	}
	group.Wait()
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
	restarted := &DiskSource{Directory: d}
	count, err := restarted.Count("tenant-one", "subject-one")
	if err != nil || count != 1 {
		t.Fatalf("cross-process replay executed %d times: %v", count, err)
	}
}
