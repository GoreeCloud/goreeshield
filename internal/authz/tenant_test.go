package authz

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGuardTenantAndSubjectMustMatchVerifiedClaims(t *testing.T) {
	now := time.Date(2026, 10, 25, 13, 0, 0, 0, time.UTC)
	claims := Claims{
		Issuer: "synthetic-issuer", Audience: "synthetic-audience",
		Tenant: "tenant-one", Subject: "subject-one",
		Scopes: []string{"privacy.synthetic.execute"},
		NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
	}
	guard := Guard{
		Verifier: fixtureVerifier{claims: claims},
		ExpectedIssuer: "synthetic-issuer", ExpectedAudience: "synthetic-audience",
		Now: func() time.Time { return now },
	}
	if err := guard.CheckTenant(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", "privacy.synthetic.execute"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tenant string
		subject string
		scope string
	}{
		{"tenant-two", "subject-one", "privacy.synthetic.execute"},
		{"", "subject-one", "privacy.synthetic.execute"},
		{"tenant-one", "subject-two", "privacy.synthetic.execute"},
		{"tenant-one", "subject-one", "privacy.synthetic.admin"},
	} {
		if err := guard.CheckTenant(context.Background(), "synthetic-fixture", tc.tenant, tc.subject, tc.scope); !errors.Is(err, ErrDenied) {
			t.Fatalf("unsafe tenant or subject claims accepted: %+v %v", tc, err)
		}
	}
	claims.Tenant = ""
	guard.Verifier = fixtureVerifier{claims: claims}
	if err := guard.CheckTenant(context.Background(), "synthetic-fixture", "tenant-one", "subject-one", "privacy.synthetic.execute"); !errors.Is(err, ErrDenied) {
		t.Fatal("missing trusted tenant claim must deny", err)
	}
}
