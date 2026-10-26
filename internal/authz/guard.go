// Package authz defines a fail-closed verifier boundary for FUTURE
// GoreeCloud Identity integration. No production token verifier is shipped.
package authz

import (
	"context"
	"errors"
	"time"
)

var ErrDenied = errors.New("identity authorization unavailable or denied")

// Claims must come from a trusted verifier, never a request-body assertion.
// This struct is not itself cryptographic proof or an authorization token.
type Claims struct {
	Issuer    string
	Audience  string
	Subject   string
	Tenant    string
	Scopes    []string
	NotBefore time.Time
	ExpiresAt time.Time
}

// Verifier will be implemented by the actual GoreeCloud Identity integration
// after issuer, audience, signature, key rotation, revocation, and token type
// semantics are reviewed. The prototype intentionally provides none.
type Verifier interface {
	Verify(ctx context.Context, opaqueCredential string) (Claims, error)
}

type Guard struct {
	Verifier         Verifier
	ExpectedIssuer   string
	ExpectedAudience string
	Now              func() time.Time
}

// Check denies by default unless a trusted verifier returns exact matching
// issuer, audience, subject, scope and a valid time window. No delegation,
// wildcard scope, or multi-subject authority exists in this experiment.
func (g Guard) Check(ctx context.Context, credential, subject, requiredScope string) error {
	return g.check(ctx, credential, "", subject, requiredScope)
}

// CheckTenant binds a synthetic request to BOTH a trusted tenant claim and
// subject. The verifier is called exactly once. This is not token validation:
// no real GoreeCloud Identity verifier is shipped or configured.
func (g Guard) CheckTenant(ctx context.Context, credential, tenant, subject, requiredScope string) error {
	if tenant == "" {
		return ErrDenied
	}
	return g.check(ctx, credential, tenant, subject, requiredScope)
}

func (g Guard) check(ctx context.Context, credential, tenant, subject, requiredScope string) error {
	if ctx == nil || ctx.Err() != nil {
		return ErrDenied
	}
	if g.Verifier == nil || g.ExpectedIssuer == "" || g.ExpectedAudience == "" ||
		credential == "" || subject == "" || requiredScope == "" {
		return ErrDenied
	}
	claims, err := g.Verifier.Verify(ctx, credential)
	if err != nil || ctx.Err() != nil {
		return ErrDenied
	}
	now := time.Now().UTC()
	if g.Now != nil {
		now = g.Now().UTC()
	}
	if now.IsZero() || claims.Issuer != g.ExpectedIssuer ||
		claims.Audience != g.ExpectedAudience || claims.Subject != subject ||
		(tenant != "" && claims.Tenant != tenant) ||
		claims.NotBefore.IsZero() || claims.ExpiresAt.IsZero() ||
		now.Before(claims.NotBefore) || !now.Before(claims.ExpiresAt) {
		return ErrDenied
	}
	for _, scope := range claims.Scopes {
		if scope == requiredScope {
			return nil
		}
	}
	return ErrDenied
}
