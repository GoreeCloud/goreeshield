//go:build linux

package choices

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSerializedOptionalAction(t *testing.T) {
	st := Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	counter := 0
	fn := func() error { counter++; return nil }
	ok, err := st.WithAllowed("demo-local-metrics", fn)
	if err != nil || ok || counter != 0 {
		t.Fatalf("default was not denied: %v %v %d", ok, err, counter)
	}
	if err := st.Update(func(s *State) error { return s.Grant("demo-local-metrics", "1", true, fixed) }); err != nil {
		t.Fatal(err)
	}
	ok, err = st.WithAllowed("demo-local-metrics", fn)
	if err != nil || !ok || counter != 1 {
		t.Fatalf("grant not recognized: %v %v %d", ok, err, counter)
	}
	if err := st.Update(func(s *State) error { return s.Withdraw("demo-local-metrics", fixed.Add(time.Minute)) }); err != nil {
		t.Fatal(err)
	}
	ok, err = st.WithAllowed("demo-local-metrics", fn)
	if err != nil || ok || counter != 1 {
		t.Fatalf("revocation failed: %v %v %d", ok, err, counter)
	}
}
