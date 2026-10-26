//go:build linux

// Package sourcehead models the owning source's side of a signed head
// protocol. It is NOT a GoreeShield signing service or production daemon.
package sourcehead

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"errors"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
)

var ErrSignerUnavailable = errors.New("source-owned signer unavailable")
var ErrSourceIdentity = errors.New("source-owned signing identity differs from durable head")
var ErrSignerKeyMismatch = errors.New("source signer public key differs from the independently pinned key")

// Responder reads source-owned durable history for each challenge and signs
// with a caller-injected signer. The verifier must provision public-key trust
// independently. This type never saves signing material to disk.
type Responder struct {
	Store  checkpoint.DiskHeadStore
	Signer crypto.Signer
	Issuer string
	KeyID  string
	// PinnedPublicKey must be provisioned outside the mutable head store and
	// outside the key-loading path. A missing pin refuses all signing.
	PinnedPublicKey ed25519.PublicKey
}

// Respond signs the current source head. A failure to read persistent
// history must not be replaced by a cached or invented checkpoint.
// Access to this method must be independently authorized by the source.
func (r Responder) Respond(ctx context.Context, request checkpoint.HeadRequest) (checkpoint.HeadProof, error) {
	if ctx == nil || ctx.Err() != nil {
		return checkpoint.HeadProof{}, checkpoint.ErrHeadUnavailable
	}
	if r.Signer == nil {
		return checkpoint.HeadProof{}, ErrSignerUnavailable
	}
	pub, ok := r.Signer.Public().(ed25519.PublicKey)
	if !ok || len(pub) != ed25519.PublicKeySize {
		return checkpoint.HeadProof{}, ErrSignerUnavailable
	}
	if len(r.PinnedPublicKey) != ed25519.PublicKeySize ||
		subtle.ConstantTimeCompare(pub, r.PinnedPublicKey) != 1 {
		return checkpoint.HeadProof{}, ErrSignerKeyMismatch
	}
	if request.ScopeDigest == ([32]byte{}) || request.Nonce == ([32]byte{}) {
		return checkpoint.HeadProof{}, checkpoint.ErrInvalid
	}
	statement, err := r.Store.Read(request.ScopeDigest)
	if err != nil {
		return checkpoint.HeadProof{}, err
	}
	if ctx.Err() != nil {
		return checkpoint.HeadProof{}, checkpoint.ErrHeadUnavailable
	}
	if statement.Issuer != r.Issuer || statement.KeyID != r.KeyID {
		return checkpoint.HeadProof{}, ErrSourceIdentity
	}
	head := checkpoint.HeadProof{Statement: statement, Nonce: request.Nonce}
	payload, err := checkpoint.HeadSigningPayload(head)
	if err != nil {
		return checkpoint.HeadProof{}, err
	}
	signature, err := r.Signer.Sign(rand.Reader, payload, crypto.Hash(0))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return checkpoint.HeadProof{}, ErrSignerUnavailable
	}
	if !ed25519.Verify(pub, payload, signature) {
		return checkpoint.HeadProof{}, ErrSignerUnavailable
	}
	if ctx.Err() != nil {
		return checkpoint.HeadProof{}, checkpoint.ErrHeadUnavailable
	}
	head.Signature = signature
	return head, nil
}

// CurrentHead implements checkpoint.HeadProvider for local source-owned
// fixtures. A real service must separately enforce caller authorization.
func (r Responder) CurrentHead(ctx context.Context, request checkpoint.HeadRequest) (checkpoint.HeadProof, error) {
	return r.Respond(ctx, request)
}
