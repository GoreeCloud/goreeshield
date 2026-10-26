//go:build linux

package synthetic

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

func TestScopedSyntheticProcessingAndWithdrawal(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	makeClaims := func(tenant, subject string) authz.Claims {
		return authz.Claims{
			Issuer: "synthetic-issuer", Audience: "goreeshield-fixture",
			Tenant: tenant, Subject: subject, Scopes: []string{"privacy.synthetic.execute"},
			NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
		}
	}
	guardFor := func(tenant, subject string) authz.Guard {
		return authz.Guard{
			Verifier:       fixtureIdentity{claims: makeClaims(tenant, subject)},
			ExpectedIssuer: "synthetic-issuer", ExpectedAudience: "goreeshield-fixture",
			Now: func() time.Time { return now },
		}
	}
	p := &ScopedProcessor{Directory: scoped.Directory{Root: filepath.Join(t.TempDir(), "scoped")}}
	a, err := p.Directory.Store("tenant-one", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Directory.Store("tenant-two", "subject-one")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, now)
	}); err != nil {
		t.Fatal(err)
	}
	if err := b.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, now)
	}); err != nil {
		t.Fatal(err)
	}
	request := contract.Request{
		Version: contract.Version, OperationID: "op-001",
		PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1,
	}
	p.Guard = guardFor("tenant-one", "subject-one")
	if result, err := p.Process(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", request); err != nil || !result.Executed {
		t.Fatalf("valid first tenant failed: %+v %v", result, err)
	}
	if result, err := p.Process(context.Background(), "synthetic-fixture", "tenant-two", "subject-one", request); !errors.Is(err, authz.ErrDenied) || result.Executed {
		t.Fatalf("cross-tenant attempt accepted: %+v %v", result, err)
	}
	if result, err := p.Process(context.Background(), "synthetic-fixture", "tenant-one", "subject-two", request); !errors.Is(err, authz.ErrDenied) || result.Executed {
		t.Fatalf("cross-subject attempt accepted: %+v %v", result, err)
	}
	p.Guard = guardFor("tenant-two", "subject-one")
	if result, err := p.Process(context.Background(), "synthetic-fixture", "tenant-two", "subject-one", request); err != nil || !result.Executed {
		t.Fatalf("same operation ID across distinct tenant should execute: %+v %v", result, err)
	}
	p.Guard = guardFor("tenant-one", "subject-one")
	if err := a.Update(func(s *choices.State) error {
		return s.Withdraw("demo-local-metrics", now.Add(time.Second))
	}); err != nil {
		t.Fatal(err)
	}
	request.OperationID = "op-002"
	request.ChoiceRevision = 2
	if result, err := p.Process(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", request); err != nil || result.Executed || result.Status != "denied" {
		t.Fatalf("withdrawn tenant still executed: %+v %v", result, err)
	}
	p.Guard = guardFor("tenant-two", "subject-one")
	request.ChoiceRevision = 1
	if result, err := p.Process(context.Background(), "synthetic-fixture", "tenant-two", "subject-one", request); err != nil || !result.Executed {
		t.Fatalf("unrelated tenant affected by withdrawal: %+v %v", result, err)
	}
	if p.engines[a.Path].Sink.Count("demo-local-metrics") != 1 || p.engines[b.Path].Sink.Count("demo-local-metrics") != 2 {
		t.Fatal("synthetic sink counts were not isolated")
	}
}

func TestScopedProcessorDeniesMissingVerifierBeforeStoreAccess(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-yet-created")
	p := &ScopedProcessor{Directory: scoped.Directory{Root: root}}
	req := contract.Request{
		Version: contract.Version, OperationID: "op-001",
		PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1,
	}
	out, err := p.Process(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", req)
	if !errors.Is(err, authz.ErrDenied) || out.Executed {
		t.Fatalf("missing verifier permitted access: %+v %v", out, err)
	}
	if len(p.engines) != 0 {
		t.Fatal("unauthorized request populated scoped processor map")
	}
}
