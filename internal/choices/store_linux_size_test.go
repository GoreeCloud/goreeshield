//go:build linux

package choices

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// A valid initial private choice must not be destroyed when an otherwise
// in-memory valid edit serializes beyond the store's 1 MiB read ceiling.
func TestChoicesStoreRejectsOversizedUpdateWithoutOverwritingPriorConsent(t *testing.T) {
	store := storeForTest(t)
	if err := store.Update(func(s *State) error {
		return s.Grant("demo-local-metrics", "1", true, fixed)
	}); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	oversized := strings.Repeat("not-a-notice", 110000)
	err = store.Update(func(s *State) error {
		ch := s.Choices["demo-local-metrics"]
		ch.NoticeVersion = oversized
		s.Choices["demo-local-metrics"] = ch
		return nil
	})
	if err == nil {
		t.Fatal("oversized update was accepted despite the 1 MiB read cap")
	}
	after, err := os.ReadFile(store.Path)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatalf("rejected update changed the prior private state: %v", err)
	}
	good, err := store.Read()
	if err != nil || !good.Allowed("demo-local-metrics") {
		t.Fatalf("prior consent unavailable after rejected mutation: %v", err)
	}
}
