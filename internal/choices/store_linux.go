//go:build linux

package choices

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Store is deliberately a single-user, Linux-local prototype. A Unix account
// is the only isolation boundary; this is NOT the GoreeCloud Identity service.
type Store struct{ Path string }

func DefaultPath() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("XDG_STATE_HOME must be absolute")
	}
	return filepath.Join(base, "goreeshield", "choices.json"), nil
}

// checkDirectory walks the absolute parent from / using Linux directory FDs.
// No path component is followed through a symlink, even while creating a
// previously missing directory. The returned parent remains pinned for every
// local state/lock operation; callers must close it.
func (s Store) checkDirectory() (*os.File, error) {
	if !filepath.IsAbs(s.Path) {
		return nil, errors.New("state path must be absolute")
	}
	clean := filepath.Clean(s.Path)
	name := filepath.Base(clean)
	if name == "." || name == "/" || name == ".." {
		return nil, errors.New("state path must identify a file")
	}
	parentPath := filepath.Dir(clean)
	fd, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(fd), "/")
	for _, part := range strings.Split(strings.TrimPrefix(parentPath, "/"), "/") {
		if part == "" {
			continue
		}
		flags := syscall.O_RDONLY | syscall.O_DIRECTORY | syscall.O_NOFOLLOW | syscall.O_CLOEXEC
		next, openErr := syscall.Openat(int(current.Fd()), part, flags, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkErr := syscall.Mkdirat(int(current.Fd()), part, 0700); mkErr != nil && !errors.Is(mkErr, syscall.EEXIST) {
				current.Close()
				return nil, mkErr
			}
			next, openErr = syscall.Openat(int(current.Fd()), part, flags, 0)
		}
		if openErr != nil {
			current.Close()
			return nil, errors.New("unsafe state directory ancestor")
		}
		current.Close()
		current = os.NewFile(uintptr(next), part)
	}
	info, err := current.Stat()
	if err != nil {
		current.Close()
		return nil, err
	}
	if !privateDirectory(info) {
		current.Close()
		return nil, errors.New("state directory must be an owner-held private directory (0700)")
	}
	return current, nil
}

func privateDirectory(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.IsDir() && info.Mode()&os.ModeSymlink == 0 &&
		info.Mode().Perm()&0077 == 0 && stat.Uid == uint32(os.Geteuid())
}

func privateRegular(info os.FileInfo) bool {
	// A second hard link can expose a 0600 state/lock inode outside this
	// private directory. Symlink checks alone do not detect hard links.
	// Check the owner as well as the mode: a privileged process must not
	// adopt a different account's local consent file or lock.
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0 &&
		stat.Nlink == 1 && stat.Uid == uint32(os.Geteuid())
}

// withLock serializes cooperating readers and writers across CLI processes.
// All operations use the pinned private-directory handle, not a re-resolved
// absolute pathname that might follow a replaced ancestor.
func (s Store) withLock(fn func(*os.File) error) error {
	parent, err := s.checkDirectory()
	if err != nil {
		return err
	}
	defer parent.Close()
	fd, err := syscall.Openat(int(parent.Fd()), ".choices.lock",
		syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	lock := os.NewFile(uintptr(fd), ".choices.lock")
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil {
		return err
	}
	if !privateRegular(info) {
		return errors.New("lock file permissions or type are unsafe")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	// Another cooperating process must not have acquired a newly replaced
	// lock path. Malicious concurrent activity by the SAME Unix account
	// remains outside this prototype's protection contract.
	checkFD, err := syscall.Openat(int(parent.Fd()), ".choices.lock",
		syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return errors.New("state lock path changed")
	}
	visible := os.NewFile(uintptr(checkFD), ".choices.lock")
	visibleInfo, visibleErr := visible.Stat()
	visible.Close()
	if visibleErr != nil || !os.SameFile(info, visibleInfo) {
		return errors.New("state lock path changed")
	}
	return fn(parent)
}

func (s Store) readUnlocked(parent *os.File) (State, error) {
	fd, err := syscall.Openat(int(parent.Fd()), filepath.Base(filepath.Clean(s.Path)),
		syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return EmptyState(), nil
	}
	if err != nil {
		return State{}, err
	}
	f := os.NewFile(uintptr(fd), "choices.json")
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return State{}, err
	}
	if !privateRegular(info) {
		return State{}, errors.New("state file must be a private regular file (0600)")
	}
	if info.Size() > 1024*1024 {
		return State{}, errors.New("state file exceeds 1 MiB limit")
	}
	// Reject ambiguous JSON object keys before Go's normal decoder can
	// silently let a later duplicate overwrite an earlier privacy decision.
	// Always cap the actual read as well as checking the original file size.
	payload, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if err != nil {
		return State{}, err
	}
	if len(payload) > 1024*1024 {
		return State{}, errors.New("state file exceeds 1 MiB limit")
	}
	if err := rejectDuplicateJSONKeys(payload); err != nil {
		return State{}, err
	}
	var state State
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&state); err != nil {
		// Decoder errors can echo attacker-controlled JSON field names.
		// Preserve fail-closed behavior without disclosing local private keys.
		return State{}, errors.New("invalid state file")
	}
	var more any
	if err := dec.Decode(&more); !errors.Is(err, io.EOF) {
		return State{}, errors.New("state file contains trailing data")
	}
	if err := state.Validate(); err != nil {
		// State validation errors may include names loaded from the private
		// local file. Do not disclose those untrusted purpose identifiers to
		// callers or logs; reject the entire state without echoing its fields.
		return State{}, errors.New("invalid state file")
	}
	if state.Choices == nil {
		state.Choices = map[string]Choice{}
	}
	return state, nil
}

func (s Store) Read() (State, error) {
	var state State
	err := s.withLock(func(parent *os.File) error {
		var e error
		state, e = s.readUnlocked(parent)
		return e
	})
	return state, err
}

func (s Store) Update(edit func(*State) error) error {
	return s.withLock(func(parent *os.File) error {
		state, err := s.readUnlocked(parent)
		if err != nil {
			return err
		}
		if err = edit(&state); err != nil {
			return err
		}
		if err = state.Validate(); err != nil {
			// Never expose untrusted private purpose identifiers.
			return errors.New("invalid state mutation")
		}
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return err
		}
		tmp := ".choices-" + hex.EncodeToString(nonce[:]) + ".tmp"
		dirFD := int(parent.Fd())
		fd, err := syscall.Openat(dirFD, tmp,
			syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
		if err != nil {
			return err
		}
		// Removal always addresses this pinned private directory, even if
		// a pathname ancestor is subsequently replaced.
		defer syscall.Unlinkat(dirFD, tmp)
		out := os.NewFile(uintptr(fd), tmp)
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err = enc.Encode(state); err != nil {
			out.Close()
			return err
		}
		if err = out.Sync(); err != nil {
			out.Close()
			return err
		}
		if err = out.Close(); err != nil {
			return err
		}
		if err = syscall.Renameat(dirFD, tmp, dirFD, filepath.Base(filepath.Clean(s.Path))); err != nil {
			return err
		}
		return parent.Sync()
	})
}
