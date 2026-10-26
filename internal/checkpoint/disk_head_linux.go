//go:build linux

package checkpoint

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

var ErrHeadAbsent = errors.New("source head history absent")
var ErrHeadConflict = errors.New("source head sequence conflict")
var ErrHeadRegression = errors.New("source head sequence regressed")
var ErrHeadGap = errors.New("source head sequence skipped")
var ErrHeadCorrupt = errors.New("source head history corrupt")
var ErrHeadCommitUncertain = errors.New("source head publication uncertain")

const maxHeadBytes = 4096

type DiskHeadStore struct {
	Root string
}

type storedHead struct {
	Version int `json:"version"`
	Statement Statement `json:"statement"`
}

func (s DiskHeadStore) path(scope [32]byte) string {
	return filepath.Join(s.Root, "head-"+hex.EncodeToString(scope[:])+".json")
}
