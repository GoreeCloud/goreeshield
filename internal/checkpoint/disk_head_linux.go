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

func readStoredHead(path string, scope [32]byte) (Statement, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return Statement{}, ErrHeadAbsent
	}
	if err != nil {
		return Statement{}, err
	}
	defer f.Close()
	if err := privateHeadFile(f); err != nil {
		return Statement{}, err
	}
	info, err := f.Stat()
	if err != nil || info.Size() > maxHeadBytes {
		return Statement{}, ErrHeadCorrupt
	}
	decoder := json.NewDecoder(io.LimitReader(f, maxHeadBytes+1))
	decoder.DisallowUnknownFields()
	var data storedHead
	if err := decoder.Decode(&data); err != nil {
		return Statement{}, ErrHeadCorrupt
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Statement{}, ErrHeadCorrupt
	}
	if data.Version != 1 || data.Statement.ScopeDigest != scope {
		return Statement{}, ErrHeadCorrupt
	}
	if _, err := SigningPayload(data.Statement); err != nil {
		return Statement{}, ErrHeadCorrupt
	}
	return data.Statement, nil
}

// Read never invents a source-owned current head if state is absent.
func (s DiskHeadStore) Read(scope [32]byte) (Statement, error) {
	var statement Statement
	err := s.withLock(scope, func(path string) error {
		var readErr error
		statement, readErr = readStoredHead(path, scope)
		return readErr
	})
	return statement, err
}

const initializedHeadMarker = "goreeshield-owned-source-head-v1\n"

func validateHeadMarker(path string) (bool, error) {
	f, err := os.OpenFile(path+".initialized", os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	if err := privateHeadFile(f); err != nil {
		return false, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(len(initializedHeadMarker)+1)))
	if err != nil || string(raw) != initializedHeadMarker {
		return false, ErrHeadCorrupt
	}
	return true, nil
}

func initializeHeadMarker(path string) error {
	exists, err := validateHeadMarker(path)
	if err != nil {
		return err
	}
	if exists {
		return ErrHeadConflict
	}
	file, err := os.OpenFile(path+".initialized", os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(initializedHeadMarker); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
