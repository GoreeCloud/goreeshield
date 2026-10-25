package authz

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fixtureVerifier struct {
	claims Claims
	err    error
}

func (f fixtureVerifier) Verify(_ context.Context, credential string) (Claims, error) {
	if credential != "synthetic-fixture" || f.err != nil {
		return Claims{}, ErrDenied
	}
	return f.claims, nil
}

func TestGuardValidatesEveryClaimDimension(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	valid := Claims{
		Issuer: "synthetic-issuer", Audience: "goreeshield-fixture",
		Subject: "subject-a", Scopes: []string{"privacy.synthetic.execute"},
		NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
	}
	base := Guard{
		Verifier: fixtureVerifier{claims: valid},
		ExpectedIssuer: "synthetic-issuer",
		ExpectedAudience: "goreeshield-fixture",
		Now: func() time.Time { return now },
	}
	if err := base.Check(context.Background(), "synthetic-fixture", "subject-a", "privacy.synthetic.execute"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		change func(*Claims)
	}{
		{"wrong-issuer", func(c *Claims) { c.Issuer = "untrusted-issuer" }},
		{"wrong-audience", func(c *Claims) { c.Audience = "other-service" }},
		{"wrong-subject", func(c *Claims) { c.Subject = "subject-b" }},
		{"missing-scope", func(c *Claims) { c.Scopes = []string{"privacy.synthetic.read"} }},
		{"expired", func(c *Claims) { c.ExpiresAt = now }},
		{"not-yet-valid", func(c *Claims) { c.NotBefore = now.Add(time.Minute) }},
		{"no-not-before", func(c *Claims) { c.NotBefore = time.Time{} }},
		{"no-expiry", func(c *Claims) { c.ExpiresAt = time.Time{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cl := valid
			tc.change(&cl)
			guard := base
			guard.Verifier = fixtureVerifier{claims: cl}
			if err := guard.Check(context.Background(), "synthetic-fixture", "subject-a", "privacy.synthetic.execute"); !errors.Is(err, ErrDenied) {
				t.Fatalf("unsafe synthetic claims accepted: %v", err)
			}
		})
	}
}

func TestGuardFailsClosedWithoutProductionVerifier(t *testing.T) {
	g := Guard{ExpectedIssuer: "synthetic-issuer", ExpectedAudience: "goreeshield-fixture"}
	if err := g.Check(context.Background(), "fake", "subject-a", "privacy.synthetic.execute"); !errors.Is(err, ErrDenied) {
		t.Fatal("missing verifier must deny")
	}
	if err := g.Check(context.Background(), "", "subject-a", "privacy.synthetic.execute"); !errors.Is(err, ErrDenied) {
		t.Fatal("missing credential must deny")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := g.Check(ctx, "fake", "subject-a", "privacy.synthetic.execute"); !errors.Is(err, ErrDenied) {
		t.Fatal("canceled request must deny")
	}
}
