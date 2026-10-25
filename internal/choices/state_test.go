package choices

import (
	"testing"
	"time"
)

var fixed = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func TestDefaultDenyAndExplicitGrant(t *testing.T) {
	s := EmptyState()
	if s.Allowed("demo-local-metrics") || s.Allowed("unknown") {
		t.Fatal("a missing choice must deny")
	}
	if err := s.Grant("demo-local-metrics", "1", false, fixed); err == nil {
		t.Fatal("grant without acknowledgement")
	}
	if err := s.Grant("demo-local-metrics", "older", true, fixed); err == nil {
		t.Fatal("stale notice accepted")
	}
	if err := s.Grant("unknown", "1", true, fixed); err == nil {
		t.Fatal("unknown purpose accepted")
	}
	if err := s.Grant("demo-local-metrics", "1", true, fixed); err != nil {
		t.Fatal(err)
	}
	if !s.Allowed("demo-local-metrics") {
		t.Fatal("recorded consent should allow only this example")
	}
	if s.Allowed("demo-remote-assistance") {
		t.Fatal("grant leaked into another purpose")
	}
	if err := s.Withdraw("demo-local-metrics", fixed.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if s.Allowed("demo-local-metrics") {
		t.Fatal("withdrawal must deny")
	}
}

func TestStaleNoticeAndInvalidState(t *testing.T) {
	s := EmptyState()
	if err := s.Grant("demo-remote-assistance", "1", true, fixed); err != nil {
		t.Fatal(err)
	}
	s.Choices["demo-remote-assistance"] = Choice{Granted: true, NoticeVersion: "0", ChangedAt: fixed}
	if s.Allowed("demo-remote-assistance") {
		t.Fatal("stale notice must deny")
	}
	s.SchemaVersion = 77
	if err := s.Validate(); err == nil {
		t.Fatal("unknown schema silently accepted")
	}
}
