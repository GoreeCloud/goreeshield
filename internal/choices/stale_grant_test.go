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
	// Reversing the clock must not roll back the last known ordering
	// watermark. Otherwise an old grant could arrive after withdrawal
	// carrying a timestamp that appears newer than the skewed withdrawal.
	if got := state.Choices["demo-local-metrics"].ChangedAt; !got.Equal(start) {
		t.Fatalf("withdrawal watermark moved backward: got %v, want %v", got, start)
	}
	if state.Revision != 2 {
		t.Fatalf("withdrawal did not consume exactly one revision: %d", state.Revision)
	}
	for _, replay := range []time.Time{start.Add(-500 * time.Millisecond), start} {
		if err := state.Grant("demo-local-metrics", "1", true, replay); err == nil {
			t.Fatal("grant older than pre-withdrawal watermark must remain denied")
		}
		if state.Allowed("demo-local-metrics") || state.Revision != 2 {
			t.Fatal("stale replay modified withdrawn consent or revision")
		}
	}
	// Repeated revocations during backward clock skew are accepted and
	// cannot lower the watermark to an earlier recorded event time.
	if err := state.Withdraw("demo-local-metrics", start.Add(-2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := state.Choices["demo-local-metrics"].ChangedAt; !got.Equal(start) {
		t.Fatalf("repeated withdrawal rolled back logical time: %v", got)
	}
	if state.Revision != 3 || state.Allowed("demo-local-metrics") {
		t.Fatal("repeated withdrawal must remain an effective denial")
	}
}
