//go:build linux

package scoped

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

func TestScopePathsAreDistinctStableAndPrivate(t *testing.T) {
	d := Directory{Root: filepath.Join(t.TempDir(), "scoped")}
	a, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	aAgain, err := d.Store("tenant-one", "subject-one")
	if err != nil || a.Path != aAgain.Path {
		t.Fatalf("unstable scoped path: %+v %+v %v", a, aAgain, err)
	}
	b, err := d.Store("tenant-two", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	c, err := d.Store("tenant-one", "subject-two")
	if err != nil {
		t.Fatal(err)
	}
	if a.Path == b.Path || a.Path == c.Path || b.Path == c.Path {
		t.Fatal("tenant/subject paths collided")
	}
	if strings.Contains(a.Path, "tenant-one") || strings.Contains(a.Path, "subject-one") {
		t.Fatal("raw identifiers leaked into scoped filename")
	}
	if info, err := os.Stat(d.Root); err != nil || info.Mode().Perm()&0077 != 0 {
		t.Fatalf("scoped directory is not private: %v %v", info, err)
	}
	if err := a.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	stateA, err := a.Read()
	if err != nil || !stateA.Allowed("demo-local-metrics") {
		t.Fatal("first subject grant was not recorded", err)
	}
	for _, other := range []choices.Store{b, c} {
		state, err := other.Read()
		if err != nil || state.Allowed("demo-local-metrics") || state.Revision != 0 {
			t.Fatal("grant leaked into different tenant or subject", err)
		}
	}
}

func TestScopeRejectsBadIDsAndUnsafeDirectory(t *testing.T) {
	d := Directory{Root: filepath.Join(t.TempDir(), "scoped")}
	for _, tc := range [][2]string{
		{"", "subject-one"}, {"a", "subject-one"},
		{"tenant-one", "../escape"}, {"tenant-one", "Subject-Upper"},
		{"tenant-one", "subject.with.dots"}, {"tenant-one", ""},
	} {
		if _, err := d.Store(tc[0], tc[1]); !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("bad identifiers accepted: %q %q %v", tc[0], tc[1], err)
		}
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := (Directory{Root: filepath.Join(root, "alias")}).Store("tenant-one", "subject-one"); err == nil {
		t.Fatal("symlinked root was accepted")
	}
	open := filepath.Join(root, "open")
	if err := os.Mkdir(open, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := (Directory{Root: open}).Store("tenant-one", "subject-one"); err == nil {
		t.Fatal("world-readable root was accepted")
	}
	if _, err := (Directory{Root: "relative"}).Store("tenant-one", "subject-one"); err == nil {
		t.Fatal("relative root was accepted")
	}
}
