//go:build linux

package reconcile

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/contract"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

var errSyntheticAcknowledgement = errors.New("test acknowledgement unavailable")

type acknowledgementFixture struct {
	source *DiskSource
	before bool
}

func (f acknowledgementFixture) Apply(ctx context.Context, tenant, subject string, req contract.Request) error {
	if f.before {
		return errSyntheticAcknowledgement
	}
	if err := f.source.Apply(ctx, tenant, subject, req); err != nil {
		return err
	}
	return errSyntheticAcknowledgement
}

func (f acknowledgementFixture) Inspect(ctx context.Context, tenant, subject, id string) (Observation, error) {
	return f.source.Inspect(ctx, tenant, subject, id)
}

func TestSyntheticAcknowledgementMissingAfterSourceCommit(t *testing.T) {
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
	source := &DiskSource{Directory: d}
	coord := Coordinator{
		Guard:  fixtureGuard("privacy.synthetic.execute", "privacy.synthetic.inspect"),
		Source: acknowledgementFixture{source: source},
	}
	req := fixtureRequest("op-ack-missing", 1)
	dispatched, err := coord.Dispatch(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if !errors.Is(err, errSyntheticAcknowledgement) || dispatched.Status != "unresolved" {
		t.Fatalf("lost acknowledgement was treated as success: %+v %v", dispatched, err)
	}
	observed, err := coord.Reconcile(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || observed.Status != "source_observed_applied" {
		t.Fatalf("committed source record not recovered: %+v %v", observed, err)
	}
	if err := (&DiskSource{Directory: d}).Apply(context.Background(), "tenant-one", "subject-one", req); err != nil {
		t.Fatal(err)
	}
	count, err := source.Count("tenant-one", "subject-one")
	if err != nil || count != 1 {
		t.Fatalf("retry repeated synthetic action: %d %v", count, err)
	}
}

func TestSyntheticAcknowledgementMissingBeforeSourceCommit(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	source := &DiskSource{Directory: d}
	coord := Coordinator{
		Guard:  fixtureGuard("privacy.synthetic.execute", "privacy.synthetic.inspect"),
		Source: acknowledgementFixture{source: source, before: true},
	}
	req := fixtureRequest("op-no-commit", 1)
	dispatched, err := coord.Dispatch(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if !errors.Is(err, errSyntheticAcknowledgement) || dispatched.Status != "unresolved" {
		t.Fatalf("no-commit submission reported success: %+v %v", dispatched, err)
	}
	observed, err := coord.Reconcile(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || observed.Status != "unresolved" {
		t.Fatalf("absence was interpreted as confirmation: %+v %v", observed, err)
	}
}
