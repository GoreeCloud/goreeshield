package choices

import (
	"testing"
	"time"
)

func TestLegacyChoiceNeverReactivatesWhenOtherPurposeChanges(t *testing.T) {
	at := time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)
	s := State{
		SchemaVersion: 1,
		Choices: map[string]Choice{
			"demo-local-metrics": {Granted: true, NoticeVersion: "1", ChangedAt: at},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if s.Allowed("demo-local-metrics") {
		t.Fatal("legacy grant unexpectedly enabled")
	}
	if err := s.Grant("demo-remote-assistance", "1", true, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !s.Allowed("demo-remote-assistance") || s.Allowed("demo-local-metrics") {
		t.Fatal("another purpose revived a legacy opt-in")
	}
	if s.Choices["demo-local-metrics"].Revision != 0 {
		t.Fatal("legacy grant revision was silently rewritten")
	}
	if err := s.Withdraw("demo-remote-assistance", at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if s.Allowed("demo-local-metrics") {
		t.Fatal("unrelated withdrawal revived a legacy opt-in")
	}
	if err := s.Grant("demo-local-metrics", "1", true, at.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if s.Revision != 3 || s.Choices["demo-local-metrics"].Revision != 3 || !s.Allowed("demo-local-metrics") {
		t.Fatal("explicit regrant did not bind the correct revision")
	}
}

func TestPriorPrototypeGrantsFailClosedAndFutureRevisionRejected(t *testing.T) {
	now := time.Now().UTC()
	s := State{
		SchemaVersion: 1,
		Revision:      7,
		Choices: map[string]Choice{
			"demo-local-metrics": {Granted: true, NoticeVersion: "1", ChangedAt: now},
		},
	}
	if s.Allowed("demo-local-metrics") {
		t.Fatal("unversioned grant with global revision must not be trusted")
	}
	if err := s.Grant("demo-remote-assistance", "1", true, now); err != nil {
		t.Fatal(err)
	}
	if s.Allowed("demo-local-metrics") {
		t.Fatal("previously persisted grant revived on unrelated update")
	}
	bad := s.Choices["demo-local-metrics"]
	bad.Revision = s.Revision + 1
	s.Choices["demo-local-metrics"] = bad
	if err := s.Validate(); err == nil {
		t.Fatal("future-revision forged grant accepted")
	}
}
