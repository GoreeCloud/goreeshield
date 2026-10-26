//go:build linux

package reconcile

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

func TestDiskSourceSurvivesRestartAndDuplicateDelivery(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	req := fixtureRequest("op-001", 1)
	first := &DiskSource{Directory: d}
	c := Coordinator{Guard: fixtureGuard("privacy.synthetic.execute", "privacy.synthetic.inspect"), Source: first}
	pending, err := c.Dispatch(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || pending.Status != "pending_source_readback" {
		t.Fatalf("dispatch must stay pending: %+v %v", pending, err)
	}
	// New fixture objects simulate a process restart. No prior map survives.
	second := &DiskSource{Directory: d}
	c.Source = second
	confirmed, err := c.Reconcile(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || confirmed.Status != "source_observed_applied" {
		t.Fatalf("persisted observation missing: %+v %v", confirmed, err)
	}
	for i := 0; i < 3; i++ {
		if err := second.Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
			t.Fatal(err)
		}
	}
	count, err := second.Count("tenant-one", "subject-one")
	if err != nil || count != 1 {
		t.Fatalf("synthetic effect replayed: %d %v", count, err)
	}
	altered := req
	altered.ChoiceRevision = 2
	if _, err := c.Reconcile(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", altered); err != nil {
		t.Fatal(err)
	}
}

func TestDiskSourceConcurrentInstancesAtMostOnce(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	a := &DiskSource{Directory: d}
	b := &DiskSource{Directory: d}
	req := fixtureRequest("op-concurrent", 1)
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 12; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); errs <- a.Apply(context.Background(), "tenant-one", "subject-one", req) }()
		go func() { defer wg.Done(); errs <- b.Apply(context.Background(), "tenant-one", "subject-one", req) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	count, err := (&DiskSource{Directory: d}).Count("tenant-one", "subject-one")
	if err != nil || count != 1 {
		t.Fatalf("duplicate source effect: %d %v", count, err)
	}
}
