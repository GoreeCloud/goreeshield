//go:build linux

package reconcile

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/authz"
	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/contract"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

type fixtureVerifier struct{ claims authz.Claims }

func (f fixtureVerifier) Verify(_ context.Context, credential string) (authz.Claims, error) {
	if credential != "synthetic-fixture" {
		return authz.Claims{}, authz.ErrDenied
	}
	return f.claims, nil
}

func fixtureGuard(scopes ...string) authz.Guard {
	at := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	return authz.Guard{
		Verifier: fixtureVerifier{claims: authz.Claims{
			Issuer: "synthetic-issuer", Audience: "goreeshield-fixture",
			Tenant: "tenant-one", Subject: "subject-one", Scopes: scopes,
			NotBefore: at.Add(-time.Minute), ExpiresAt: at.Add(time.Minute),
		}},
		ExpectedIssuer: "synthetic-issuer", ExpectedAudience: "goreeshield-fixture",
		Now: func() time.Time { return at },
	}
}

func fixtureRequest(id string, revision uint64) contract.Request {
	return contract.Request{Version: contract.Version, OperationID: id, PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: revision}
}

func TestPendingUntilSourceReadbackAndRetryIsIdempotent(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "private")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	source := &MockSource{Directory: d}
	c := Coordinator{Guard: fixtureGuard("privacy.synthetic.execute", "privacy.synthetic.inspect"), Source: source}
	req := fixtureRequest("op-001", 1)
	ctx := context.Background()
	out, err := c.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || out.Status != "pending_source_readback" {
		t.Fatalf("submission falsely confirmed: %+v %v", out, err)
	}
	count, err := source.Count("tenant-one", "subject-one")
	if err != nil || count != 1 {
		t.Fatalf("synthetic action not recorded: %d %v", count, err)
	}
	out, err = c.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || out.Status != "pending_source_readback" {
		t.Fatalf("retry should remain pending: %+v %v", out, err)
	}
	count, _ = source.Count("tenant-one", "subject-one")
	if count != 1 {
		t.Fatalf("retry executed again: count %d", count)
	}
	out, err = c.Reconcile(ctx, "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || out.Status != "source_observed_applied" {
		t.Fatalf("source did not confirm: %+v %v", out, err)
	}
	unknown, err := c.Reconcile(ctx, "synthetic-fixture", "tenant-one", "subject-one", fixtureRequest("op-unknown", 1))
	if err != nil || unknown.Status != "unresolved" {
		t.Fatalf("missing observation must be unresolved: %+v %v", unknown, err)
	}
}

func TestWithdrawalAndConflictsNeverBecomeApplied(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "private")}
	store, err := d.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	source := &MockSource{Directory: d}
	c := Coordinator{Guard: fixtureGuard("privacy.synthetic.execute", "privacy.synthetic.inspect"), Source: source}
	if err := store.Update(func(s *choices.State) error {
		return s.Withdraw("demo-local-metrics", time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	req := fixtureRequest("op-denied", 1)
	ctx := context.Background()
	pending, err := c.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || pending.Status != "pending_source_readback" {
		t.Fatalf("denied attempt still needs readback: %+v %v", pending, err)
	}
	observed, err := c.Reconcile(ctx, "synthetic-fixture", "tenant-one", "subject-one", req)
	if err != nil || observed.Status != "source_observed_denied" {
		t.Fatalf("denial not observed: %+v %v", observed, err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-one", req); err != nil {
		t.Fatal("identical retry must remain duplicate", err)
	}
	collision := req
	collision.ChoiceRevision = 2
	if _, err := c.Dispatch(ctx, "synthetic-fixture", "tenant-one", "subject-one", collision); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed payload with same operation id should conflict: %v", err)
	}
	conflict, err := c.Reconcile(ctx, "synthetic-fixture", "tenant-one", "subject-one", collision)
	if err != nil || conflict.Status != "conflict" {
		t.Fatalf("collision reconciliation must conflict: %+v %v", conflict, err)
	}
	count, err := source.Count("tenant-one", "subject-one")
	if err != nil || count != 0 {
		t.Fatalf("withdrawn/duplicate source should not execute: %d %v", count, err)
	}
}
