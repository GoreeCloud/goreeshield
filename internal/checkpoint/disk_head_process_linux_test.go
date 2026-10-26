//go:build linux

package checkpoint

import (
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
