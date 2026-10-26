package checkpoint

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"testing"
)

func fixture(t *testing.T) (Proof, Trust, Expected, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	scope := sha256.Sum256([]byte("synthetic scope fixture"))
	journal := sha256.Sum256([]byte("synthetic journal fixture"))
	statement := Statement{
		Version: Version,
		Issuer: "goreecloud-fixture-source",
		KeyID: "fixture-key-v1",
		ScopeDigest: scope,
		Sequence: 7,
		JournalDigest: journal,
	}
	payload, err := SigningPayload(statement)
	if err != nil {
		t.Fatal(err)
	}
	return Proof{Statement: statement, Signature: ed25519.Sign(priv, payload)},
		Trust{Issuer: statement.Issuer, KeyID: statement.KeyID, PublicKey: pub},
		Expected{ScopeDigest: scope, MinimumSequence: 7, JournalDigest: journal}, priv
}

func resign(t *testing.T, p *Proof, priv ed25519.PrivateKey) {
	t.Helper()
	body, err := SigningPayload(p.Statement)
	if err != nil {
		t.Fatal(err)
	}
	p.Signature = ed25519.Sign(priv, body)
}

func TestTrustedSourceCheckpoint(t *testing.T) {
	proof, trust, expected, _ := fixture(t)
	if err := Verify(proof, trust, expected); err != nil {
		t.Fatalf("signed current source checkpoint was rejected: %v", err)
	}
}

func TestOlderValidSignatureRejectedByTrustedSourceHead(t *testing.T) {
	proof, trust, expected, priv := fixture(t)
	proof.Statement.Sequence = 6
	resign(t, &proof, priv)
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrStale) {
		t.Fatalf("authentic but old checkpoint was accepted: %v", err)
	}
}

func TestCrossScopeAndWrongJournalRejected(t *testing.T) {
	proof, trust, expected, priv := fixture(t)
	proof.Statement.ScopeDigest = sha256.Sum256([]byte("other tenant"))
	resign(t, &proof, priv)
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrMismatch) {
		t.Fatalf("cross-scope checkpoint accepted: %v", err)
	}
	proof, trust, expected, priv = fixture(t)
	proof.Statement.JournalDigest = sha256.Sum256([]byte("rolled-back journal"))
	resign(t, &proof, priv)
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrMismatch) {
		t.Fatalf("mismatched journal checkpoint accepted: %v", err)
	}
}

func TestWrongSignerAndModifiedStatementRejected(t *testing.T) {
	proof, trust, expected, _ := fixture(t)
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	trust.PublicKey = pub
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrSignature) {
		t.Fatalf("wrong public key accepted: %v", err)
	}
	proof, trust, expected, _ = fixture(t)
	proof.Statement.Sequence++
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrSignature) {
		t.Fatalf("unsigned sequence change accepted: %v", err)
	}
	proof, trust, expected, _ = fixture(t)
	trust.KeyID = "unknown-key"
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrUntrusted) {
		t.Fatalf("wrong trusted key ID accepted: %v", err)
	}
}

func TestInvalidShapesAndMissingIndependentHead(t *testing.T) {
	proof, trust, expected, _ := fixture(t)
	expected.MinimumSequence = 0
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing trusted sequence minimum accepted: %v", err)
	}
	proof, trust, expected, _ = fixture(t)
	proof.Signature = proof.Signature[:8]
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrSignature) {
		t.Fatalf("truncated signature accepted: %v", err)
	}
	proof, trust, expected, _ = fixture(t)
	trust.PublicKey = trust.PublicKey[:8]
	if err := Verify(proof, trust, expected); !errors.Is(err, ErrUntrusted) {
		t.Fatalf("truncated public key accepted: %v", err)
	}
	for _, change := range []func(*Statement){
		func(s *Statement) { s.Version = 2 },
		func(s *Statement) { s.Sequence = 0 },
		func(s *Statement) { s.KeyID = "unsafe:key" },
		func(s *Statement) { s.Issuer = "" },
		func(s *Statement) { s.ScopeDigest = [32]byte{} },
		func(s *Statement) { s.JournalDigest = [32]byte{} },
	} {
		proof, trust, expected, _ = fixture(t)
		change(&proof.Statement)
		if err := Verify(proof, trust, expected); !errors.Is(err, ErrInvalid) {
			t.Fatalf("malformed checkpoint statement accepted: %v", err)
		}
	}
}
