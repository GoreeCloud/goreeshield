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
