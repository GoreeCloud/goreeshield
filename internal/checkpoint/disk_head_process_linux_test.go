//go:build linux

package checkpoint

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Child mode is used only by a short-lived local Linux test subprocess.
func TestHeadStoreSubprocessWorker(t *testing.T) {
	if os.Getenv("GS_SOURCE_HEAD_CHILD") != "1" {
		return
	}
	store := DiskHeadStore{Root: os.Getenv("GS_SOURCE_HEAD_ROOT")}
	if err := store.Advance(headStatement(3)); err != nil {
		t.Fatal(err)
	}
	head, err := store.Read(headScope())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("HEADSEQ=%d\n", head.Sequence)
}

func TestDiskHeadConcurrentLinuxProcesses(t *testing.T) {
	root := filepath.Join(t.TempDir(), "source-owned")
	store := DiskHeadStore{Root: root}
	if err := store.Initialize(headStatement(1)); err != nil {
		t.Fatal(err)
	}
	if err := store.Advance(headStatement(2)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	children := make([]*exec.Cmd, 2)
	outputs := make([]*bytes.Buffer, len(children))
	for i := range children {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHeadStoreSubprocessWorker$")
		cmd.Env = append(os.Environ(), "GS_SOURCE_HEAD_CHILD=1", "GS_SOURCE_HEAD_ROOT="+root)
		outputs[i] = new(bytes.Buffer)
		cmd.Stdout = outputs[i]
		cmd.Stderr = outputs[i]
		children[i] = cmd
	}
	for _, cmd := range children {
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range children {
		err := cmd.Wait()
		output := outputs[i].String()
		if err != nil || !strings.Contains(output, "HEADSEQ=3") {
			t.Fatalf("independent source-head worker failed: %v %q", err, output)
		}
	}
	restarted := DiskHeadStore{Root: root}
	head, err := restarted.Read(headScope())
	if err != nil || head != headStatement(3) {
		t.Fatalf("source history did not survive process writes: %+v %v", head, err)
	}
}
