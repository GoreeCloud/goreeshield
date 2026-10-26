//go:build linux

package reconcile

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/authz"
	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

func TestSeparateInspectionScopeAndSubjects(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "private")}
	source := &MockSource{Directory: d}
	req := fixtureRequest("op-isolated", 1)
	ctx := context.Background()

	onlyExecute := Coordinator{Guard: fixtureGuard("privacy.synthetic.execute"), Source: source}
	if _, err := onlyExecute.Reconcile(ctx, "synthetic-fixture", "tenant-one", "subject-one", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("inspection without scoped privilege was allowed: %v", err)
	}
	onlyInspect := Coordinator{Guard: fixtureGuard("privacy.synthetic.inspect"), Source: source}
	if _, err := onlyInspect.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-one", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("execution without scoped privilege was allowed: %v", err)
	}
	if _, err := onlyExecute.Dispatch(ctx, "synthetic-fixture", "tenant-two", "subject-one", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("wrong tenant accepted: %v", err)
	}
	if _, err := onlyExecute.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-two", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("wrong subject accepted: %v", err)
	}
	if _, err := onlyExecute.Dispatch(ctx, "wrong-credential", "tenant-one", "subject-one", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("unverified credential accepted: %v", err)
	}
	canceled, stop := context.WithCancel(ctx)
	stop()
	if _, err := onlyExecute.Dispatch(canceled, "synthetic-fixture", "tenant-one", "subject-one", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("canceled operation accepted: %v", err)
	}
	if _, err := source.Inspect(ctx, "tenant-one", "subject-one", req.OperationID); err != nil {
		t.Fatal(err)
	}
}

func TestSourceScopedOperationIDsAndConcurrentRetry(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "private")}
	source := &MockSource{Directory: d}
	for _, tenant := range []string{"tenant-one", "tenant-two"} {
		store, err := d.Store(tenant, "subject-one")
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Update(func(s *choices.State) error {
			return s.Grant("demo-local-metrics", "1", true, time.Now())
		}); err != nil {
			t.Fatal(err)
		}
	}
	req := fixtureRequest("op-same", 1)
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 12; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- source.Apply(context.Background(), "tenant-one", "subject-one", req)
		}()
		go func() {
			defer wg.Done()
			errs <- source.Apply(context.Background(), "tenant-two", "subject-one", req)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tenant := range []string{"tenant-one", "tenant-two"} {
		count, err := source.Count(tenant, "subject-one")
		if err != nil || count != 1 {
			t.Fatalf("%s executed %d times: %v", tenant, count, err)
		}
		obs, err := source.Inspect(context.Background(), tenant, "subject-one", req.OperationID)
		if err != nil || !obs.Found || obs.Status != "applied" {
			t.Fatalf("source record missing: %+v %v", obs, err)
		}
	}
	absent, err := source.Inspect(context.Background(), "tenant-one", "subject-two", req.OperationID)
	if err != nil || absent.Found {
		t.Fatalf("cross-subject observation leaked: %+v %v", absent, err)
	}
}

func TestFullSourceJournalRefusesNewOperations(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "private")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	source := &MockSource{Directory: d}
	history := make(map[string]record)
	for i := 0; i < MaxSourceRecords; i++ {
		history[string(rune(i+1000))] = record{}
	}
	source.records = map[string]map[string]record{store.Path: history}
	if err := source.Apply(context.Background(), "tenant-one", "subject-one", fixtureRequest("op-full", 1)); !errors.Is(err, ErrFull) {
		t.Fatalf("journal exhaustion should fail closed: %v", err)
	}
}
