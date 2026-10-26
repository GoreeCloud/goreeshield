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

func (s DiskHeadStore) privateRoot() error {
	if !filepath.IsAbs(s.Root) || filepath.Clean(s.Root) != s.Root {
		return ErrHeadCorrupt
	}
	if err := os.MkdirAll(s.Root, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(s.Root)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 ||
		!ok || stat.Uid != uint32(os.Geteuid()) {
		return ErrHeadCorrupt
	}
	return nil
}
