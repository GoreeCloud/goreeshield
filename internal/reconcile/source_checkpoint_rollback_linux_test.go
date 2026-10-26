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

// This demonstrates a SAME-UID rollback of the whole local source tuple.
// Only a separately trusted newer sequence can reject the old signed state.
func TestExternallyTrustedHeadDetectsCoordinatedLocalRollback(t *testing.T) {
	dir := scoped.Directory{Root: filepath.Join(t.TempDir(), "scopes")}
	ctx := context.Background()
	tenant, subject := "tenant-one", "subject-one"
	source := &DiskSource{Directory: dir}
	if err := source.Apply(ctx, tenant, subject, fixtureRequest("signed-rollback-op-1", 1)); err != nil {
		t.Fatal(err)
	}
	store, err := dir.Store(tenant, subject)
	if err != nil {
		t.Fatal(err)
	}
	path := sourceLedgerPath(store.Path)
	paths := []string{path, journalMarkerPath(path), journalWitnessPath(path)}
	prior := make([][]byte, len(paths))
	for i, path := range paths {
		prior[i], err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := checkpoint.ScopeDigest(tenant, subject)
	if err != nil {
		t.Fatal(err)
	}
	statement := checkpoint.Statement{
		Version: checkpoint.Version,
		Issuer: "external-source-fixture",
		KeyID: "ephemeral-test-key",
		ScopeDigest: scope,
		Sequence: 10,
		JournalDigest: sha256.Sum256(prior[0]),
	}
	payload, err := checkpoint.SigningPayload(statement)
	if err != nil {
		t.Fatal(err)
	}
	proof := checkpoint.Proof{Statement: statement, Signature: ed25519.Sign(priv, payload)}
	trust := checkpoint.Trust{Issuer: statement.Issuer, KeyID: statement.KeyID, PublicKey: pub}
	if err := source.Apply(ctx, tenant, subject, fixtureRequest("signed-rollback-op-2", 1)); err != nil {
		t.Fatal(err)
	}
	for i, path := range paths {
		if err := os.WriteFile(path, prior[i], 0600); err != nil {
			t.Fatal(err)
		}
	}
	// When all local files move backwards together, local consistency
	// checks alone cannot distinguish the old state from a current state.
	if _, err := source.Count(tenant, subject); err != nil {
		t.Fatalf("coordinated local rollback fixture was not valid: %v", err)
	}
	if err := VerifySyntheticJournalCheckpoint(ctx, dir, tenant, subject, proof, trust, 10); err != nil {
		t.Fatalf("old signed state should validate with a stale floor: %v", err)
	}
	if err := VerifySyntheticJournalCheckpoint(ctx, dir, tenant, subject, proof, trust, 11); !errors.Is(err, checkpoint.ErrStale) {
		t.Fatalf("independent newer sequence failed to reject old signed state: %v", err)
	}
}
