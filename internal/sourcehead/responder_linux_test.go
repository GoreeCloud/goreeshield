//go:build linux

package sourcehead

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
)

func testStatement(sequence uint64) checkpoint.Statement {
	return checkpoint.Statement{
		Version: checkpoint.Version, Issuer: "owning-fixture", KeyID: "key-v1",
		ScopeDigest:   sha256.Sum256([]byte("synthetic-scope")),
		Sequence:      sequence,
		JournalDigest: sha256.Sum256([]byte("synthetic-journal")),
	}
}

func testProof(t *testing.T, s checkpoint.Statement, priv ed25519.PrivateKey) checkpoint.Proof {
	t.Helper()
	payload, err := checkpoint.SigningPayload(s)
	if err != nil {
		t.Fatal(err)
	}
	return checkpoint.Proof{Statement: s, Signature: ed25519.Sign(priv, payload)}
}

func TestResponderReadsLatestDurableHead(t *testing.T) {
	store := checkpoint.DiskHeadStore{Root: filepath.Join(t.TempDir(), "heads")}
	first := testStatement(1)
	if err := store.Initialize(first); err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	responder := Responder{Store: store, Signer: private, Issuer: first.Issuer, KeyID: first.KeyID, PinnedPublicKey: public}
	trust := checkpoint.Trust{Issuer: first.Issuer, KeyID: first.KeyID, PublicKey: public}
	proof := testProof(t, first, private)
	ctx := context.Background()
	if err := checkpoint.VerifyWithFreshHead(ctx, responder, proof, trust, first.ScopeDigest, first.JournalDigest); err != nil {
		t.Fatalf("initial signed head rejected: %v", err)
	}
	if err := store.Advance(testStatement(2)); err != nil {
		t.Fatal(err)
	}
	if err := checkpoint.VerifyWithFreshHead(ctx, responder, proof, trust, first.ScopeDigest, first.JournalDigest); !errors.Is(err, checkpoint.ErrHeadStale) {
		t.Fatalf("old signature accepted after head advance: %v", err)
	}
	restarted := responder
	restarted.Store = checkpoint.DiskHeadStore{Root: store.Root}
	out, err := restarted.Respond(ctx, checkpoint.HeadRequest{ScopeDigest: first.ScopeDigest, Nonce: sha256.Sum256([]byte("challenge"))})
	if err != nil || out.Statement.Sequence != 2 {
		t.Fatalf("head not recovered after reopen: %+v %v", out, err)
	}
}

func TestResponderFailsClosedOnUntrustedOrMissingState(t *testing.T) {
	store := checkpoint.DiskHeadStore{Root: filepath.Join(t.TempDir(), "heads")}
	first := testStatement(1)
	request := checkpoint.HeadRequest{ScopeDigest: first.ScopeDigest, Nonce: sha256.Sum256([]byte("nonce"))}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	responder := Responder{Store: store, Signer: private, Issuer: first.Issuer, KeyID: first.KeyID, PinnedPublicKey: public}
	if _, err := responder.Respond(context.Background(), request); !errors.Is(err, checkpoint.ErrHeadAbsent) {
		t.Fatalf("missing source head accepted: %v", err)
	}
	if err := store.Initialize(first); err != nil {
		t.Fatal(err)
	}
	responder.Signer = nil
	if _, err := responder.Respond(context.Background(), request); !errors.Is(err, ErrSignerUnavailable) {
		t.Fatalf("missing signer accepted: %v", err)
	}
	responder.Signer = private
	responder.KeyID = "wrong-key"
	if _, err := responder.Respond(context.Background(), request); !errors.Is(err, ErrSourceIdentity) {
		t.Fatalf("wrong signing identity accepted: %v", err)
	}
	responder.KeyID = first.KeyID
	request.Nonce = [32]byte{}
	if _, err := responder.Respond(context.Background(), request); !errors.Is(err, checkpoint.ErrInvalid) {
		t.Fatalf("empty nonce accepted: %v", err)
	}
}
