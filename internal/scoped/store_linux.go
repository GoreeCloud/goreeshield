//go:build linux

// Package scoped maps synthetic tenant/subject pairs to separate local choice
// files. It is NOT a production multitenant storage or OS isolation system.
package scoped

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

var ErrInvalidScope = errors.New("invalid synthetic tenant or subject")

// Directory is a private Linux directory containing one choice store per
// synthetic tenant/subject pair. All records still share one Unix account.
type Directory struct {
	Root string
}

func validateID(id string) bool {
	if len(id) < 3 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func (d Directory) checkRoot() error {
	if !filepath.IsAbs(d.Root) || filepath.Clean(d.Root) != d.Root {
		return errors.New("scoped root must be a clean absolute path")
	}
	// Refuse symlinks in existing ancestors as well as the root itself.
	for current := d.Root; ; current = filepath.Dir(current) {
		fi, err := os.Lstat(current)
		if err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				return errors.New("scoped directory ancestry must not contain symlinks")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if current == string(filepath.Separator) {
			break
		}
	}
	if err := os.MkdirAll(d.Root, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(d.Root)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return errors.New("scoped root is not a private directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("scoped root is not owned by current Linux user")
	}
	return nil
}

// Store derives a stable local path without including raw subject/tenant in
// filenames. The deterministic digest is NOT encryption or anonymization.
func (d Directory) Store(tenant, subject string) (choices.Store, error) {
	if !validateID(tenant) || !validateID(subject) || strings.ContainsAny(tenant+subject, "/\\") {
		return choices.Store{}, ErrInvalidScope
	}
	if err := d.checkRoot(); err != nil {
		return choices.Store{}, err
	}
	digest := sha256.Sum256([]byte("goreeshield-synthetic-scope-v1\x00" + tenant + "\x00" + subject))
	name := "scope-" + hex.EncodeToString(digest[:]) + ".json"
	return choices.Store{Path: filepath.Join(d.Root, name)}, nil
}
