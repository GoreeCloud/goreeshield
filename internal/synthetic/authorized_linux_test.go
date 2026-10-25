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
)

type fixtureIdentity struct {
	claims authz.Claims
}

func (v fixtureIdentity) Verify(_ context.Context, credential string) (authz.Claims, error) {
	if credential != "synthetic-fixture" {
		return authz.Claims{}, authz.ErrDenied
	}
	return v.claims, nil
}

func TestGuardedSyntheticProcessorRejectsImpersonationAndRevocation(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	store := choices.Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	if err := store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, now)
	}); err != nil {
		t.Fatal(err)
	}
	sink := &Sink{}
	engine := &Processor{Store: store, Sink: sink}
	claims := authz.Claims{
		Issuer: "synthetic-issuer", Audience: "goreeshield-fixture",
		Subject: "subject-a", Scopes: []string{"privacy.synthetic.execute"},
		NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
	}
	makeGuard := func(c authz.Claims) authz.Guard {
		return authz.Guard{
			Verifier: fixtureIdentity{claims: c},
			ExpectedIssuer: "synthetic-issuer",
			ExpectedAudience: "goreeshield-fixture",
			Now: func() time.Time { return now },
		}
	}
	proxy := GuardedProcessor{Guard: makeGuard(claims), Engine: engine, LocalSubject: "subject-a"}
	req := contract.Request{
		Version: contract.Version, OperationID: "op-001",
		PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1,
	}
	if _, err := proxy.Process(context.Background(), "synthetic-fixture", "subject-b", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("other subject must be denied: %v", err)
	}
	wrongSubject := claims
	wrongSubject.Subject = "subject-b"
	proxy.Guard = makeGuard(wrongSubject)
	if _, err := proxy.Process(context.Background(), "synthetic-fixture", "subject-a", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("mismatched verified subject was accepted: %v", err)
	}
	proxy.Guard = makeGuard(claims)
	if _, err := proxy.Process(context.Background(), "wrong-credential", "subject-a", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("unverified credential accepted: %v", err)
	}
	if sink.Count("demo-local-metrics") != 0 {
		t.Fatal("failed authorization caused synthetic action")
	}
	out, err := proxy.Process(context.Background(), "synthetic-fixture", "subject-a", req)
	if err != nil || !out.Executed || out.Status != "applied" || sink.Count("demo-local-metrics") != 1 {
		t.Fatalf("synthetic permitted action failed: %+v %v", out, err)
	}
	if err := store.Update(func(s *choices.State) error {
		return s.Withdraw("demo-local-metrics", now.Add(time.Second))
	}); err != nil {
		t.Fatal(err)
	}
	req.OperationID = "op-002"
	req.ChoiceRevision = 2
	out, err = proxy.Process(context.Background(), "synthetic-fixture", "subject-a", req)
	if err != nil || out.Executed || out.Status != "denied" || sink.Count("demo-local-metrics") != 1 {
		t.Fatalf("withdrawal did not deny: %+v %v", out, err)
	}
	proxy.Guard.Verifier = nil
	if _, err := proxy.Process(context.Background(), "synthetic-fixture", "subject-a", req); !errors.Is(err, authz.ErrDenied) {
		t.Fatal("no verifier must deny", err)
	}
}
