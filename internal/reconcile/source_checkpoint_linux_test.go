//go:build linux

package reconcile

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

func TestSignedCheckpointBindsCurrentSyntheticJournal(t *testing.T) {
	d := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	ctx := context.Background()
	source := &DiskSource{Directory: d}
	tenant, subject := "tenant-one", "subject-one"
	if err := source.Apply(ctx, tenant, subject, fixtureRequest("checkpoint-op-1", 1)); err != nil {
		t.Fatal(err)
	}
	store, err := d.Store(tenant, subject)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := os.ReadFile(sourceLedgerPath(store.Path))
	if err != nil {
		t.Fatal(err)
	}
	scopeHash, err := checkpoint.ScopeDigest(tenant, subject)
	if err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	statement := checkpoint.Statement{
		Version: checkpoint.Version,
		Issuer: "external-source-fixture",
		KeyID: "ephemeral-test-key",
		ScopeDigest: scopeHash,
		Sequence: 10,
		JournalDigest: sha256.Sum256(journal),
	}
	payload, err := checkpoint.SigningPayload(statement)
	if err != nil {
		t.Fatal(err)
	}
	proof := checkpoint.Proof{Statement: statement, Signature: ed25519.Sign(priv, payload)}
	trust := checkpoint.Trust{Issuer: statement.Issuer, KeyID: statement.KeyID, PublicKey: pub}
	if err := VerifySyntheticJournalCheckpoint(ctx, d, tenant, subject, proof, trust, 10); err != nil {
		t.Fatalf("matching external fixture checkpoint rejected: %v", err)
	}
	if err := VerifySyntheticJournalCheckpoint(ctx, d, tenant, subject, proof, trust, 11); !errors.Is(err, checkpoint.ErrStale) {
		t.Fatalf("signed checkpoint older than trusted head accepted: %v", err)
	}
	if err := source.Apply(ctx, tenant, subject, fixtureRequest("checkpoint-op-2", 1)); err != nil {
		t.Fatal(err)
	}
	if err := VerifySyntheticJournalCheckpoint(ctx, d, tenant, subject, proof, trust, 10); !errors.Is(err, checkpoint.ErrMismatch) {
		t.Fatalf("old signed journal state was accepted after new source write: %v", err)
	}
}

func TestSignedCheckpointCannotCrossSyntheticScopes(t *testing.T) {
	a, err := checkpoint.ScopeDigest("tenant-a", "subject-x")
	if err != nil {
		t.Fatal(err)
	}
	b, err := checkpoint.ScopeDigest("tenant-b", "subject-x")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("synthetic tenant scope digest was not isolated")
	}
	c, err := checkpoint.ScopeDigest("tenant-a", "subject-y")
	if err != nil {
		t.Fatal(err)
	}
	if a == c {
		t.Fatal("synthetic subject scope digest was not isolated")
	}
}
