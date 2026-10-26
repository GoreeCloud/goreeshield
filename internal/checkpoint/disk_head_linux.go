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

func privateHeadFile(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 ||
		!ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 {
		return ErrHeadCorrupt
	}
	return nil
}

func (s DiskHeadStore) withLock(scope [32]byte, fn func(string) error) error {
	if emptyDigest(scope) {
		return ErrInvalid
	}
	if err := s.privateRoot(); err != nil {
		return err
	}
	path := s.path(scope)
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := privateHeadFile(lock); err != nil {
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn(path)
}
