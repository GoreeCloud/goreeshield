//go:build linux

package choices

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistedLegacyGrantIsNotRestoredByAnotherGrant(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	if _, err := store.Read(); err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`{"schema_version":1,"revision":4,"choices":{"demo-local-metrics":{"granted":true,"notice_version":"1","changed_at":"2026-10-25T17:00:00Z"}}}`)
	if err := os.WriteFile(store.Path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := store.Read()
	if err != nil || before.Allowed("demo-local-metrics") {
		t.Fatalf("legacy file should load but deny: %+v %v", before, err)
	}
	if err := store.Update(func(s *State) error {
		return s.Grant("demo-remote-assistance", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	after, err := store.Read()
	if err != nil || after.Revision != 5 || after.Allowed("demo-local-metrics") {
		t.Fatalf("unrelated write revived a legacy grant: %+v %v", after, err)
	}
	executed := false
	ok, err := store.WithMatchingRevision("demo-local-metrics", "1", after.Revision, func() error {
		executed = true
		return nil
	})
	if err != nil || ok || executed {
		t.Fatalf("revision gate accepted a legacy grant: %t %t %v", ok, executed, err)
	}
	if err := store.Update(func(s *State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	renewed, err := store.Read()
	if err != nil || !renewed.Allowed("demo-local-metrics") || renewed.Choices["demo-local-metrics"].Revision != renewed.Revision {
		t.Fatalf("renewed grant invalid: %+v %v", renewed, err)
	}
}
