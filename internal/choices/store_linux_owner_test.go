//go:build linux

package choices

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Different-inode-owner metadata is simulated without privileged chown,
// ensuring this negative regression can run on ordinary CI workers.
type differentOwnerInfo struct {
	os.FileInfo
}

func (i differentOwnerInfo) Sys() any {
	original, ok := i.FileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	changed := *original
	changed.Uid = uint32(os.Geteuid()) + 1
	return &changed
}

func TestPrivateChoicesRegularFileRequiresCurrentOwner(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "choices.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !privateRegular(info) {
		t.Fatal("valid owner-only file refused")
	}
	if privateRegular(differentOwnerInfo{info}) {
		t.Fatal("private file with a different owning principal was accepted")
	}
}

func TestPrivateChoicesDirectoryRequiresCurrentOwner(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !privateDirectory(info) {
		t.Fatal("valid private directory refused")
	}
	if privateDirectory(differentOwnerInfo{info}) {
		t.Fatal("private directory with a different owning principal was accepted")
	}
}
