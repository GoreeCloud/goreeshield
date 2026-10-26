//go:build linux

package reconcile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// journalRequiresWitness reads the versioned initialization marker. V1
// permits the pre-witness migration window; V2 fails closed if the witness
// is absent. Neither version is secure against a malicious same-UID actor.
func journalRequiresWitness(path string) (bool, error) {
	marked, err := journalMarked(path)
	if err != nil || !marked {
		return false, err
	}
	file, err := os.OpenFile(journalMarkerPath(path), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return false, err
	}
	defer file.Close()
	if err := ownedPrivateRegular(file); err != nil {
		return false, fmt.Errorf("%w: %v", ErrJournalMarker, err)
	}
	raw, err := io.ReadAll(io.LimitReader(file, int64(len(witnessRequiredMarkerContents)+1)))
	if err != nil {
		return false, err
	}
	if string(raw) == journalMarkerContents {
		return false, nil
	}
	if string(raw) != witnessRequiredMarkerContents {
		return false, ErrJournalMarker
	}
	return true, nil
}

// requireJournalWitness is called only after a witness matching the new
// journal has been synchronized. It never repairs absent source history.
func requireJournalWitness(path string) error {
	required, err := journalRequiresWitness(path)
	if err != nil || required {
		return err
	}
	want, found, err := readJournalWitness(path)
	if err != nil {
		return err
	}
	if !found {
		return ErrJournalWitness
	}
	got, err := hashCurrentJournal(path)
	if err != nil {
		return err
	}
	if want != got {
		return ErrJournalRollback
	}
	parent := filepath.Dir(path)
	temp, err := os.CreateTemp(parent, ".marker-*.tmp")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.WriteString(witnessRequiredMarkerContents); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, journalMarkerPath(path)); err != nil {
		return err
	}
	dir, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// Presence of the marker and witness only detects accidental loss of one
// file; deletion or rollback of the whole same-UID tuple is not prevented.
var ErrWitnessMissing = errors.New("initialized journal requires a checksum witness")
