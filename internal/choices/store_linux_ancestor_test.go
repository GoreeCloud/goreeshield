//go:build linux

package choices

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestChoicesStoreRejectsSymlinkAncestorBeforeCreatingState(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	shortcut := filepath.Join(root, "shortcut")
	if err := os.Symlink(destination, shortcut); err != nil {
		t.Fatal(err)
	}
	store := Store{Path: filepath.Join(shortcut, "new-private", "choices.json")}
	if _, err := store.Read(); err == nil {
		t.Fatal("followed a symlinked ancestor")
	}
	if _, err := os.Stat(filepath.Join(destination, "new-private")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created an unintended directory through symlink: %v", err)
	}
}
