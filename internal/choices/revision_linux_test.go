//go:build linux

package choices

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRevisionGateSerializesWithdrawal(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	if err := store.Update(func(s *State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	actionDone := make(chan error, 1)
	go func() {
		_, err := store.WithMatchingRevision("demo-local-metrics", "1", 1, func() error {
			close(entered)
			<-release
			return nil
		})
		actionDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("gate did not start")
	}
	withdrawDone := make(chan error, 1)
	go func() {
		withdrawDone <- store.Update(func(s *State) error {
			return s.Withdraw("demo-local-metrics", time.Now())
		})
	}()
	select {
	case err := <-withdrawDone:
		t.Fatalf("withdrawal overtook in-progress gated action: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-actionDone:
		if err != nil { t.Fatal(err) }
	case <-time.After(3 * time.Second):
		t.Fatal("gated action timeout")
	}
	select {
	case err := <-withdrawDone:
		if err != nil { t.Fatal(err) }
	case <-time.After(3 * time.Second):
		t.Fatal("withdrawal timeout")
	}
	ran := false
	allowed, err := store.WithMatchingRevision("demo-local-metrics", "1", 1, func() error {
		ran = true
		return nil
	})
	if err != nil || allowed || ran {
		t.Fatalf("stale local action was not denied: %t %t %v", allowed, ran, err)
	}
}
