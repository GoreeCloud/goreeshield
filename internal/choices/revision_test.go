package choices

import (
	"testing"
	"time"
)

func TestChoiceRevisionMonotonicAndLegacyDefaultDeny(t *testing.T) {
	s := EmptyState()
	if s.Revision != 0 || s.Allowed("demo-local-metrics") {
		t.Fatal("fresh choice revision must be zero and denied")
	}
	legacy := State{SchemaVersion: 1, Choices: map[string]Choice{
		"demo-local-metrics": {Granted: true, NoticeVersion: "1", ChangedAt: time.Now().UTC()},
	}}
	if legacy.Allowed("demo-local-metrics") {
		t.Fatal("unversioned legacy grant must fail closed")
	}
	if err := legacy.Grant("demo-local-metrics", "1", true, time.Now()); err != nil { t.Fatal(err) }
	if legacy.Revision != 1 || !legacy.Allowed("demo-local-metrics") {
		t.Fatal("fresh affirmative choice must advance revision")
	}
	if err := legacy.Withdraw("demo-local-metrics", time.Now()); err != nil { t.Fatal(err) }
	if legacy.Revision != 2 || legacy.Allowed("demo-local-metrics") {
		t.Fatal("withdrawal must increment revision and deny")
	}
	if err := legacy.Grant("demo-local-metrics", "1", true, time.Now()); err != nil { t.Fatal(err) }
	if legacy.Revision != 3 { t.Fatal("regrant must not reuse stale revision") }
	legacy.Revision = ^uint64(0)
	if err := legacy.Withdraw("demo-local-metrics", time.Now()); err == nil {
		t.Fatal("revision rollover should fail closed")
	}
}
