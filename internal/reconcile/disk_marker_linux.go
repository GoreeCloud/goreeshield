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
