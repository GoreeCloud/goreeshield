//go:build linux

package reconcile

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// TestJournalWriterChild is run only in a child Go test process. It stops at
// a private writer checkpoint until its test parent terminates that process.
func TestJournalWriterChild(t *testing.T) {
	if os.Getenv("GS_JOURNAL_CHILD") != "1" {
		return
	}
	root := os.Getenv("GS_JOURNAL_ROOT")
	target := os.Getenv("GS_JOURNAL_CHECKPOINT")
	if root == "" || (target != "before_rename" && target != "after_rename") {
		t.Fatal("invalid synthetic subprocess fixture")
	}
	dir := scoped.Directory{Root: root}
	store, err := dir.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	req := fixtureRequest("op-process-interruption", 1)
	next := diskLedger{Version: 1, Count: 1, Records: map[string]record{
		req.OperationID: {Request: req, Status: "applied"},
	}}
	err = writeSourceLedgerWithCheckpoint(sourceLedgerPath(store.Path), next, func(stage string) error {
		if stage == target {
			if _, err := fmt.Fprintln(os.Stdout, "CHECKPOINT:"+stage); err != nil {
				return err
			}
			time.Sleep(30 * time.Second)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal("synthetic child was not interrupted")
}

// TestJournalWriterKilledAtCheckpoints runs a real Linux child process on
// private fixture files. It is NOT a power-loss or real-source effects test.
func TestJournalWriterKilledAtCheckpoints(t *testing.T) {
	for _, stage := range []string{"before_rename", "after_rename"} {
		t.Run(stage, func(t *testing.T) {
			dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
			store, err := dir.Store("tenant-one", "subject-one")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Update(func(s *choices.State) error {
				return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
			}); err != nil {
				t.Fatal(err)
			}
			path := sourceLedgerPath(store.Path)
			initial := diskLedger{Version: 1, Records: make(map[string]record)}
			if err := writeSourceLedger(path, initial); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestJournalWriterChild$")
			child.Env = append(os.Environ(),
				"GS_JOURNAL_CHILD=1",
				"GS_JOURNAL_ROOT="+dir.Root,
				"GS_JOURNAL_CHECKPOINT="+stage,
			)
			stdout, err := child.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			line, readErr := bufio.NewReader(stdout).ReadString('\n')
			cancel()
			exitErr := child.Wait()
			if readErr != nil || strings.TrimSpace(line) != "CHECKPOINT:"+stage {
				t.Fatalf("child did not reach target checkpoint: %q, read=%v, exit=%v", line, readErr, exitErr)
			}
			if exitErr == nil {
				t.Fatal("child unexpectedly completed without interruption")
			}
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				t.Fatal("checkpoint reached only after deadline")
			}

			recovered, err := readSourceLedger(path)
			if err != nil {
				t.Fatalf("fresh reader rejected journal: %v", err)
			}
			req := fixtureRequest("op-process-interruption", 1)
			if stage == "before_rename" {
				if recovered.Count != 0 || len(recovered.Records) != 0 {
					t.Fatalf("unpublished operation replaced journal: %+v", recovered)
				}
			} else if recovered.Count != 1 || recovered.Records[req.OperationID].Request != req {
				t.Fatalf("renamed operation not visible for readback: %+v", recovered)
			}
			source := &DiskSource{Directory: dir}
			if err := source.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
				t.Fatal(err)
			}
			count, err := source.Count("tenant-one", "subject-one")
			if err != nil || count != 1 {
				t.Fatalf("recovering retry executed synthetic operation twice: %d %v", count, err)
			}
			observation, err := source.Inspect(context.Background(), "tenant-one", "subject-one", req.OperationID)
			if err != nil || !observation.Found || observation.Status != "applied" {
				t.Fatalf("source observation missing after recovery: %+v %v", observation, err)
			}
		})
	}
}
