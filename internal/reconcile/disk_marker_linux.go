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

var ErrJournalMissing = errors.New("initialized source journal missing")
var ErrJournalMarker = errors.New("source journal marker invalid")

const journalMarkerContents = "goreeshield-synthetic-journal-v1\n"

func journalMarkerPath(path string) string { return path + ".initialized" }

func journalMarked(path string) (bool, error) {
	f, err := os.OpenFile(journalMarkerPath(path), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	if err := ownedPrivateRegular(f); err != nil {
		return false, fmt.Errorf("%w: %v", ErrJournalMarker, err)
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(len(journalMarkerContents)+1)))
	if err != nil || string(data) != journalMarkerContents {
		return false, ErrJournalMarker
	}
	return true, nil
}

// initializeJournalMarker runs before publication of a first journal file.
// If the marker survives but the journal does not, recovery fails closed.
func initializeJournalMarker(path string) error {
	exists, err := journalMarked(path)
	if err != nil {
		return err
	}
	if exists {
		_, err := readSourceLedger(path)
		return err
	}
	f, err := os.OpenFile(journalMarkerPath(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if errors.Is(err, os.ErrExist) {
		exists, err := journalMarked(path)
		if err != nil {
			return err
		}
		if exists {
			_, err := readSourceLedger(path)
			return err
		}
		return ErrJournalMarker
	}
	if err != nil {
		return err
	}
	if _, err := f.WriteString(journalMarkerContents); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
