package choices

import (
	"reflect"
	"testing"
	"time"
)

func TestStaleRegrantAfterWithdrawalCannotResurrectSyntheticConsent(t *testing.T) {
	start := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	state := EmptyState()
	if err := state.Grant("demo-local-metrics", "1", true, start); err != nil {
		t.Fatal(err)
	}
	if err := state.Withdraw("demo-local-metrics", start.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if state.Allowed("demo-local-metrics") {
		t.Fatal("withdrawal must immediately deny the optional purpose")
	}
	before := State{
		SchemaVersion: state.SchemaVersion,
		Revision:      state.Revision,
		Choices:       make(map[string]Choice, len(state.Choices)),
	}
	for purpose, choice := range state.Choices {
		before.Choices[purpose] = choice
	}
	for _, replayTime := range []time.Time{start, start.Add(time.Minute), start.Add(2 * time.Minute)} {
		if err := state.Grant("demo-local-metrics", "1", true, replayTime); err == nil {
			t.Fatal("stale or equal-time grant must be rejected")
		}
		if !reflect.DeepEqual(state, before) {
			t.Fatal("rejected stale grant must not mutate consent state or revision")
		}
		if state.Allowed("demo-local-metrics") {
			t.Fatal("stale grant unexpectedly re-enabled optional processing")
		}
	}
	// Each purpose has its own decision ordering; unrelated historical
	// fixture time must not mutate or authorize the withdrawn purpose.
	if err := state.Grant("demo-remote-assistance", "1", true, start.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !state.Allowed("demo-remote-assistance") || state.Allowed("demo-local-metrics") {
		t.Fatal("unrelated purpose decisions must remain isolated")
	}
	if err := state.Grant("demo-local-metrics", "1", true, start.Add(3*time.Minute)); err != nil {
		t.Fatal("an explicitly new timestamped grant should remain possible:", err)
	}
	if !state.Allowed("demo-local-metrics") {
		t.Fatal("new synthetic consent should authorize only after a valid fresh grant")
	}
}

func TestStaleWithdrawalRemainsAcceptedToProtectPrivacy(t *testing.T) {
	// Revoke must not be rejected solely because a local clock moved
	// backward. Source-authenticated ordering remains a release blocker.
	start := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	state := EmptyState()
	if err := state.Grant("demo-local-metrics", "1", true, start); err != nil {
		t.Fatal(err)
	}
	if err := state.Withdraw("demo-local-metrics", start.Add(-time.Second)); err != nil {
		t.Fatal("a valid withdrawal must not be blocked by timestamp ordering:", err)
	}
	if state.Allowed("demo-local-metrics") {
		t.Fatal("withdrawal must deny optional processing even with local clock skew")
	}
}
