package checkpoint

import (
	"bytes"
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrInvalid = errors.New("invalid source-owned checkpoint")
var ErrUntrusted = errors.New("source checkpoint issuer or key is not trusted")
var ErrSignature = errors.New("source checkpoint signature invalid")
var ErrMismatch = errors.New("source checkpoint scope or journal does not match")
var ErrStale = errors.New("source checkpoint sequence is below independently trusted minimum")

const Version uint16 = 1
const domain = "GoreeShield:source-checkpoint:v1\x00"

// Statement describes a remote SOURCE-owned claim about a journal digest.
// This is not evidence of data erasure or a real user rights fulfillment.
type Statement struct {
	Version uint16
	Issuer string
	KeyID string
	ScopeDigest [32]byte
	Sequence uint64
	JournalDigest [32]byte
}

type Proof struct {
	Statement Statement
	Signature []byte
}

// Trust must be provisioned independently of the local journal and witness.
// The private signing key must never be copied into GoreeShield.
type Trust struct {
	Issuer string
	KeyID string
	PublicKey ed25519.PublicKey
}

// Expected must come from a verified scope and an independently trusted
// source head sequence, not a minimum loaded only from local journal files.
type Expected struct {
	ScopeDigest [32]byte
	MinimumSequence uint64
	JournalDigest [32]byte
}

func validIdentifier(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func emptyDigest(d [32]byte) bool {
	return d == [32]byte{}
}

// SigningPayload is canonical, length-prefixed, and domain separated.
// A source implementation may sign its returned bytes with its OWN private
// key. GoreeShield's production implementation only verifies signatures.
func SigningPayload(s Statement) ([]byte, error) {
	if s.Version != Version || !validIdentifier(s.Issuer) ||
		!validIdentifier(s.KeyID) || s.Sequence == 0 ||
		emptyDigest(s.ScopeDigest) || emptyDigest(s.JournalDigest) {
		return nil, ErrInvalid
	}
	var b bytes.Buffer
	b.WriteString(domain)
	for _, id := range []string{s.Issuer, s.KeyID} {
		if err := binary.Write(&b, binary.BigEndian, uint16(len(id))); err != nil {
			return nil, fmt.Errorf("%w: identifier length: %v", ErrInvalid, err)
		}
		b.WriteString(id)
	}
	b.Write(s.ScopeDigest[:])
	if err := binary.Write(&b, binary.BigEndian, s.Sequence); err != nil {
		return nil, fmt.Errorf("%w: sequence: %v", ErrInvalid, err)
	}
	b.Write(s.JournalDigest[:])
	return b.Bytes(), nil
}

// Verify accepts only a checkpoint signed by the separately provisioned
// source public key for the expected scope and journal state. Sequence
// freshness relies on MinimumSequence from an independent authoritative
// source, not the signature alone.
func Verify(proof Proof, trust Trust, expected Expected) error {
	payload, err := SigningPayload(proof.Statement)
	if err != nil {
		return err
	}
	if !validIdentifier(trust.Issuer) || !validIdentifier(trust.KeyID) ||
		len(trust.PublicKey) != ed25519.PublicKeySize ||
		proof.Statement.Issuer != trust.Issuer || proof.Statement.KeyID != trust.KeyID {
		return ErrUntrusted
	}
	if len(proof.Signature) != ed25519.SignatureSize ||
		!ed25519.Verify(trust.PublicKey, payload, proof.Signature) {
		return ErrSignature
	}
	if emptyDigest(expected.ScopeDigest) || emptyDigest(expected.JournalDigest) ||
		subtle.ConstantTimeCompare(proof.Statement.ScopeDigest[:], expected.ScopeDigest[:]) != 1 ||
		subtle.ConstantTimeCompare(proof.Statement.JournalDigest[:], expected.JournalDigest[:]) != 1 {
		return ErrMismatch
	}
	if expected.MinimumSequence == 0 {
		return ErrInvalid
	}
	if proof.Statement.Sequence < expected.MinimumSequence {
		return ErrStale
	}
	return nil
}
