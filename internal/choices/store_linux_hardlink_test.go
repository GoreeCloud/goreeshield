//go:build linux

package choices

import (
    "bytes"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func privateLocalStateDir(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    if err := os.Chmod(dir, 0700); err != nil { t.Fatal(err) }
    return dir
}

func TestLocalChoicesRejectsHardLinkedStateBeforeReadOrUpdate(t *testing.T) {
    dir := privateLocalStateDir(t)
    external := filepath.Join(dir, "external-original.json")
    statePath := filepath.Join(dir, "choices.json")
    payload := []byte(`{"schema_version":1,"revision":0,"choices":{}}`)
    if err := os.WriteFile(external, payload, 0600); err != nil { t.Fatal(err) }
    if err := os.Link(external, statePath); err != nil { t.Fatal(err) }

    store := Store{Path:statePath}
    if _, err := store.Read(); err == nil {
        t.Fatal("state hardlink was accepted")
    } else if strings.Contains(err.Error(), "external-original") {
        t.Fatal("state rejection disclosed linked path")
    }
    called := false
    err := store.Update(func(_ *State) error {
        called = true
        return nil
    })
    if err == nil || called {
        t.Fatalf("state update accepted hardlink: called=%v err=%v", called, err)
    }
    got, err := os.ReadFile(external)
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(got, payload) { t.Fatal("external hardlinked bytes changed") }
    if err := os.Remove(statePath); err != nil { t.Fatal(err) }
    valid, err := store.Read()
    if err != nil || valid.Revision != 0 { t.Fatalf("missing state failed: %+v %v", valid, err) }
}

func TestLocalChoicesRejectsHardLinkedLock(t *testing.T) {
    dir := privateLocalStateDir(t)
    external := filepath.Join(dir, "external-lock")
    lockPath := filepath.Join(dir, ".choices.lock")
    if err := os.WriteFile(external, []byte("private"), 0600); err != nil { t.Fatal(err) }
    if err := os.Link(external, lockPath); err != nil { t.Fatal(err) }
    store := Store{Path:filepath.Join(dir,"choices.json")}
    if _, err := store.Read(); err == nil {
        t.Fatal("lock hardlink was accepted")
    }
    got, err := os.ReadFile(external)
    if err != nil { t.Fatal(err) }
    if string(got) != "private" { t.Fatal("linked lock content changed") }
    if err := os.Remove(lockPath); err != nil { t.Fatal(err) }
    if _, err := store.Read(); err != nil { t.Fatalf("normal lock failed: %v", err) }
}
