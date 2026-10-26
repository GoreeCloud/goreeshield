package checkpoint

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"testing"
)

type syntheticHeadProvider struct {
	private   ed25519.PrivateKey
	statement Statement
	failure   error
	mutate    func(*HeadProof)
	calls     int
}

func (p *syntheticHeadProvider) CurrentHead(ctx context.Context, request HeadRequest) (HeadProof, error) {
	p.calls++
	if p.failure != nil {
		return HeadProof{}, p.failure
	}
	if ctx.Err() != nil {
		return HeadProof{}, ctx.Err()
	}
	head := HeadProof{Statement: p.statement, Nonce: request.Nonce}
	if p.mutate != nil {
		p.mutate(&head)
	}
	payload, err := HeadSigningPayload(head)
	if err != nil {
		return HeadProof{}, err
	}
	head.Signature = ed25519.Sign(p.private, payload)
	return head, nil
}

func TestFreshHeadVerifiesIndependentSourceFixture(t *testing.T) {
	proof, trust, expected, private := fixture(t)
	provider := &syntheticHeadProvider{private: private, statement: proof.Statement}
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); err != nil {
		t.Fatalf("matching challenged source head rejected: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one fresh head request, got %d", provider.calls)
	}
}

func TestFreshHeadRejectsOlderSignedCheckpoint(t *testing.T) {
	proof, trust, expected, private := fixture(t)
	latest := proof.Statement
	latest.Sequence++
	provider := &syntheticHeadProvider{private: private, statement: latest}
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrHeadStale) {
		t.Fatalf("older signed checkpoint accepted as current: %v", err)
	}
}

func TestFreshHeadRejectsReplayedChallenge(t *testing.T) {
	proof, trust, expected, private := fixture(t)
	provider := &syntheticHeadProvider{
		private: private, statement: proof.Statement,
		mutate: func(head *HeadProof) { head.Nonce = sha256.Sum256([]byte("stale challenge")) },
	}
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrHeadChallenge) {
		t.Fatalf("stale challenge response accepted: %v", err)
	}
}

func TestFreshHeadRejectsOtherScopeAndJournal(t *testing.T) {
	proof, trust, expected, private := fixture(t)
	provider := &syntheticHeadProvider{private: private, statement: proof.Statement}
	provider.statement.ScopeDigest = sha256.Sum256([]byte("other synthetic scope"))
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrMismatch) {
		t.Fatalf("cross-scope source head accepted: %v", err)
	}
	provider.statement = proof.Statement
	provider.statement.JournalDigest = sha256.Sum256([]byte("newer journal"))
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrMismatch) {
		t.Fatalf("different journal head accepted: %v", err)
	}
}

func TestFreshHeadRejectsUntrustedKeyAndModifiedProof(t *testing.T) {
	proof, trust, expected, private := fixture(t)
	_, other, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider := &syntheticHeadProvider{private: other, statement: proof.Statement}
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrSignature) {
		t.Fatalf("source head from wrong private key accepted: %v", err)
	}
	proof.Statement.Sequence++
	provider.private = private
	provider.calls = 0
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrSignature) {
		t.Fatalf("modified local checkpoint accepted: %v", err)
	}
	if provider.calls != 0 {
		t.Fatal("invalid local proof triggered a source request")
	}
}

func TestFreshHeadFailsClosedWhenUnavailable(t *testing.T) {
	proof, trust, expected, private := fixture(t)
	provider := &syntheticHeadProvider{private: private, statement: proof.Statement, failure: errors.New("fixture offline")}
	if err := VerifyWithFreshHead(context.Background(), provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrHeadUnavailable) {
		t.Fatalf("offline source was accepted: %v", err)
	}
	if err := VerifyWithFreshHead(context.Background(), nil, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrHeadUnavailable) {
		t.Fatalf("missing provider was accepted: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := VerifyWithFreshHead(ctx, provider, proof, trust, expected.ScopeDigest, expected.JournalDigest); !errors.Is(err, ErrHeadUnavailable) {
		t.Fatalf("cancelled context accepted: %v", err)
	}
}
