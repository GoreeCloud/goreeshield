package choices

import (
	"testing"
	"time"
)

func TestAllowedRejectsUnsupportedStateSchema(t *testing.T) {
	s := EmptyState()
	if err := s.Grant("demo-local-metrics", "1", true, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if !s.Allowed("demo-local-metrics") {
		t.Fatal("current synthetic grant should be accepted")
	}
	s.SchemaVersion = 99
	if s.Allowed("demo-local-metrics") {
		t.Fatal("unsupported state schema must not authorize synthetic processing")
	}
	if err := s.Validate(); err == nil {
		t.Fatal("unsupported state schema should also fail validation")
	}
}

func TestChoiceMutatorsRejectUnsupportedSchema(t *testing.T) {
	for _, version := range []int{0, 2, 99} {
		s := EmptyState()
		now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
		if err := s.Grant("demo-local-metrics", "1", true, now); err != nil {
			t.Fatal(err)
		}
		original := s.Choices["demo-local-metrics"]
		originalRevision := s.Revision

		// An unsupported record must not be mutated, even by direct callers
		// that bypass the storage layer's schema validation.
		s.SchemaVersion = version
		if err := s.Grant("demo-remote-assistance", "1", true, now); err == nil {
			t.Fatalf("schema %d: grant unexpectedly accepted", version)
		}
		if err := s.Withdraw("demo-local-metrics", now.Add(time.Minute)); err == nil {
			t.Fatalf("schema %d: withdrawal unexpectedly accepted", version)
		}
		if s.Revision != originalRevision {
			t.Fatalf("schema %d: revision changed on rejected mutations", version)
		}
		if len(s.Choices) != 1 || s.Choices["demo-local-metrics"] != original {
			t.Fatalf("schema %d: consent choices changed on rejected mutations", version)
		}
	}
}
