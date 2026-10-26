package checkpoint

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"errors"
)

var ErrHeadUnavailable = errors.New("independent source head unavailable")
var ErrHeadChallenge = errors.New("source head did not answer the current challenge")
var ErrHeadStale = errors.New("signed checkpoint is not the current source head")

const headDomain = "GoreeShield:source-head-challenge:v1\x00"

// HeadRequest contains a fresh challenge for an independently hosted source.
// It must not be saved alongside GoreeShield's local journal files.
type HeadRequest struct {
	ScopeDigest [32]byte
	Nonce       [32]byte
}

// HeadProof is a source-owned signed response to a one-time challenge.
type HeadProof struct {
	Statement Statement
	Nonce     [32]byte
	Signature []byte
}

// HeadProvider is an abstract source interface, not an HTTP client or a
// production identity integration. The implementation MUST authenticate
// transport and protect its private key independently from GoreeShield.
type HeadProvider interface {
	CurrentHead(context.Context, HeadRequest) (HeadProof, error)
}

// HeadSigningPayload is a domain-separated deterministic encoding for an
// owning source to sign. GoreeShield's non-test code never signs messages.
func HeadSigningPayload(head HeadProof) ([]byte, error) {
	if emptyDigest(head.Nonce) {
		return nil, ErrInvalid
	}
	statement, err := SigningPayload(head.Statement)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString(headDomain)
	buf.Write(head.Nonce[:])
	buf.Write(statement)
	return buf.Bytes(), nil
}

// VerifyWithFreshHead confirms that a local signed checkpoint corresponds to
// a freshly challenged SOURCE-owned head, under a separately pinned key.
// A working provider, protected key, trusted source identity, and monotonic
// current sequence are deployment prerequisites and NOT created here.
func VerifyWithFreshHead(
	ctx context.Context, provider HeadProvider, proof Proof, trust Trust,
	expectedScope, currentJournal [32]byte,
) error {
	if ctx == nil || ctx.Err() != nil || provider == nil {
		return ErrHeadUnavailable
	}
	if err := Verify(proof, trust, Expected{
		ScopeDigest: expectedScope, JournalDigest: currentJournal, MinimumSequence: 1,
	}); err != nil {
		return err
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return ErrHeadUnavailable
	}
	head, err := provider.CurrentHead(ctx, HeadRequest{ScopeDigest: expectedScope, Nonce: nonce})
	if err != nil || ctx.Err() != nil {
		return ErrHeadUnavailable
	}
	if subtle.ConstantTimeCompare(head.Nonce[:], nonce[:]) != 1 {
		return ErrHeadChallenge
	}
	payload, err := HeadSigningPayload(head)
	if err != nil {
		return err
	}
	if head.Statement.Issuer != trust.Issuer ||
		head.Statement.KeyID != trust.KeyID ||
		len(trust.PublicKey) != ed25519.PublicKeySize {
		return ErrUntrusted
	}
	if len(head.Signature) != ed25519.SignatureSize ||
		!ed25519.Verify(trust.PublicKey, payload, head.Signature) {
		return ErrSignature
	}
	if subtle.ConstantTimeCompare(head.Statement.ScopeDigest[:], expectedScope[:]) != 1 {
		return ErrMismatch
	}
	if subtle.ConstantTimeCompare(head.Statement.JournalDigest[:], currentJournal[:]) != 1 {
		return ErrMismatch
	}
	if head.Statement.Sequence != proof.Statement.Sequence {
		return ErrHeadStale
	}
	return nil
}
