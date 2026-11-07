package choices

import (
	"reflect"
	"testing"
	"time"
)

func TestCorruptedStateMutatorsFailWithoutPartialChange(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		corrupt func(*State)
	}{
		{"unknown_purpose", func(s *State) {
			s.Choices["not-a-declared-purpose"] = Choice{Granted: true, NoticeVersion: "1", ChangedAt: now, Revision: s.Revision}
		}},
		{"future_decision_revision", func(s *State) {
			choice := s.Choices["demo-local-metrics"]
			choice.Revision = s.Revision + 1
			s.Choices["demo-local-metrics"] = choice
		}},
		{"missing_notice", func(s *State) {
			choice := s.Choices["demo-local-metrics"]
			choice.NoticeVersion = ""
			s.Choices["demo-local-metrics"] = choice
		}},
		{"missing_decision_time", func(s *State) {
			choice := s.Choices["demo-local-metrics"]
			choice.ChangedAt = time.Time{}
			s.Choices["demo-local-metrics"] = choice
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := EmptyState()
			if err := state.Grant("demo-local-metrics", "1", true, now); err != nil {
				t.Fatal(err)
			}
			tc.corrupt(&state)
			before := State{SchemaVersion: state.SchemaVersion, Revision: state.Revision, Choices: make(map[string]Choice)}
			for id, choice := range state.Choices {
				before.Choices[id] = choice
			}
			if err := state.Grant("demo-remote-assistance", "1", true, now.Add(time.Minute)); err == nil {
				t.Fatal("grant must reject corrupted existing state")
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatal("rejected grant mutated state")
			}
			if err := state.Withdraw("demo-local-metrics", now.Add(2*time.Minute)); err == nil {
				t.Fatal("withdraw must reject corrupted existing state")
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatal("rejected withdrawal mutated state")
			}
		})
	}
}

func TestRevisionExhaustionPreservesNilChoiceMap(t *testing.T) {
	state := State{SchemaVersion: 1, Revision: ^uint64(0)}
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	if err := state.Grant("demo-local-metrics", "1", true, now); err == nil {
		t.Fatal("grant must fail on exhausted revision counter")
	}
	if state.Choices != nil || state.Revision != ^uint64(0) {
		t.Fatal("failed grant must not initialize choices or alter revision")
	}
	if err := state.Withdraw("demo-local-metrics", now); err == nil {
		t.Fatal("withdraw must fail on exhausted revision counter")
	}
	if state.Choices != nil || state.Revision != ^uint64(0) {
		t.Fatal("failed withdrawal must not initialize choices or alter revision")
	}
}
