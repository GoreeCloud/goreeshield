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
