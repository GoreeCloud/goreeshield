//go:build linux

package choices

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Store is deliberately a single-user, Linux-local prototype. A Unix account
// is the only isolation boundary; this is NOT the GoreeCloud Identity service.
type Store struct{ Path string }

func DefaultPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil { return "", err }
		base = filepath.Join(home, ".local", "state")
	}
	if !filepath.IsAbs(base) { return "", errors.New("XDG_STATE_HOME must be absolute") }
	return filepath.Join(base, "goreeshield", "choices.json"), nil
}

func (s Store) checkDirectory() (string, error) {
	if !filepath.IsAbs(s.Path) { return "", errors.New("state path must be absolute") }
	parent := filepath.Dir(filepath.Clean(s.Path))
	if err := os.MkdirAll(parent, 0700); err != nil { return "", err }
	info, err := os.Lstat(parent)
	if err != nil { return "", err }
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return "", errors.New("state directory must be a non-symlink private directory (0700)")
	}
	return parent, nil
}

func privateRegular(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0
}

// withLock serializes cooperating readers and writers across CLI processes.
func (s Store) withLock(fn func(string) error) error {
	parent, err := s.checkDirectory()
	if err != nil { return err }
	lockPath := filepath.Join(parent, ".choices.lock")
	// O_NOFOLLOW avoids opening a symlink placed at the lock path.
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil { return err }
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil { return err }
	if !privateRegular(info) { return errors.New("lock file permissions or type are unsafe") }
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil { return err }
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn(parent)
}

func (s Store) readUnlocked() (State, error) {
	f, err := os.OpenFile(s.Path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) { return EmptyState(), nil }
	if err != nil { return State{}, err }
	defer f.Close()
	info, err := f.Stat()
	if err != nil { return State{}, err }
	if !privateRegular(info) { return State{}, errors.New("state file must be a private regular file (0600)") }
	if info.Size() > 1024*1024 { return State{}, errors.New("state file exceeds 1 MiB limit") }
	var state State
	dec := json.NewDecoder(io.LimitReader(f, 1024*1024+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&state); err != nil { return State{}, fmt.Errorf("state decode failed: %w", err) }
	var more any
	if err := dec.Decode(&more); !errors.Is(err, io.EOF) { return State{}, errors.New("state file contains trailing data") }
	if err := state.Validate(); err != nil { return State{}, err }
	if state.Choices == nil { state.Choices = map[string]Choice{} }
	return state, nil
}

func (s Store) Read() (State, error) {
	var state State
	err := s.withLock(func(_ string) error {
		var e error
		state, e = s.readUnlocked()
		return e
	})
	return state, err
}

func (s Store) Update(edit func(*State) error) error {
	return s.withLock(func(parent string) error {
		state, err := s.readUnlocked()
		if err != nil { return err }
		if err = edit(&state); err != nil { return err }
		if err = state.Validate(); err != nil { return err }
		out, err := os.CreateTemp(parent, ".choices-*.tmp")
		if err != nil { return err }
		tmp := out.Name()
		defer os.Remove(tmp)
		if err = out.Chmod(0600); err != nil { out.Close(); return err }
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err = enc.Encode(state); err != nil { out.Close(); return err }
		if err = out.Sync(); err != nil { out.Close(); return err }
		if err = out.Close(); err != nil { return err }
		if err = os.Rename(tmp, s.Path); err != nil { return err }
		dir, err := os.Open(parent)
		if err != nil { return err }
		defer dir.Close()
		return dir.Sync()
	})
}
