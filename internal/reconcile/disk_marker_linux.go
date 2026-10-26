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
