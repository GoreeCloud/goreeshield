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

// A directory pathname can change while a local choice edit is in flight.
// The pinned directory handle must retain the update inside the originally
// opened private inode, never re-resolve a replacement link.
func TestChoicesStorePinsParentAgainstReplacementDuringUpdate(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "private")
	moved := filepath.Join(root, "private-moved")
	decoy := filepath.Join(root, "decoy")
	if err := os.Mkdir(decoy, 0700); err != nil {
		t.Fatal(err)
	}
	st := Store{Path: filepath.Join(parent, "choices.json")}
	if _, err := st.Read(); err != nil {
		t.Fatal(err)
	}
	if err := st.Update(func(state *State) error {
		if err := os.Rename(parent, moved); err != nil {
			return err
		}
		if err := os.Symlink(decoy, parent); err != nil {
			return err
		}
		return state.Grant("demo-local-metrics", "1", true, fixed)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(decoy, "choices.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("state unexpectedly written to replaced path: %v", err)
	}
	kept, err := (Store{Path: filepath.Join(moved, "choices.json")}).Read()
	if err != nil || !kept.Allowed("demo-local-metrics") {
		t.Fatalf("pinned original directory did not receive edit: %v", err)
	}
	if _, err := st.Read(); err == nil {
		t.Fatal("replaced path accepted as a private state directory")
	}
}
