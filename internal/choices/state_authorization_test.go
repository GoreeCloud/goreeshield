package choices

import (
	"reflect"
	"testing"
	"time"
)

func TestAllowedDeniesValidGrantWithinCorruptUnrelatedState(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		corrupt func(*State)
	}{
		{"unknown_purpose", func(s *State) {
			s.Choices["undeclared-purpose"] = Choice{Granted: true, NoticeVersion: "1", ChangedAt: now, Revision: s.Revision}
		}},
		{"unrelated_future_revision", func(s *State) {
			s.Choices["demo-remote-assistance"] = Choice{Granted: false, NoticeVersion: "1", ChangedAt: now, Revision: s.Revision + 1}
		}},
		{"unrelated_missing_notice", func(s *State) {
			s.Choices["demo-remote-assistance"] = Choice{Granted: false, ChangedAt: now, Revision: s.Revision}
		}},
		{"unrelated_missing_timestamp", func(s *State) {
			s.Choices["demo-remote-assistance"] = Choice{Granted: false, NoticeVersion: "1", Revision: s.Revision}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := EmptyState()
			if err := s.Grant("demo-local-metrics", "1", true, now); err != nil {
				t.Fatal(err)
			}
			if !s.Allowed("demo-local-metrics") {
				t.Fatal("baseline valid synthetic grant must be allowed")
			}
			tc.corrupt(&s)
			if err := s.Validate(); err == nil {
				t.Fatal("fixture must be globally corrupt")
			}
			before := State{SchemaVersion: s.SchemaVersion, Revision: s.Revision, Choices: make(map[string]Choice)}
			for id, choice := range s.Choices {
				before.Choices[id] = choice
			}
			if s.Allowed("demo-local-metrics") {
				t.Fatal("must deny a valid isolated grant inside corrupted state")
			}
			if !reflect.DeepEqual(s, before) {
				t.Fatal("read-only permission check must not mutate state")
			}
		})
	}
}

func TestAllowedPreservesIndependentValidPurposeDecisions(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	s := EmptyState()
	if err := s.Grant("demo-local-metrics", "1", true, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Grant("demo-remote-assistance", "1", true, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !s.Allowed("demo-local-metrics") || !s.Allowed("demo-remote-assistance") {
		t.Fatal("two well-formed synthetic grants must remain independent")
	}
	if err := s.Withdraw("demo-remote-assistance", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !s.Allowed("demo-local-metrics") || s.Allowed("demo-remote-assistance") {
		t.Fatal("valid withdrawal must deny only its purpose")
	}
}
