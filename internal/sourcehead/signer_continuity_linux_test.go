//go:build linux

package sourcehead

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
)

func TestPinnedSignerSurvivesFixtureProcessRestart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "heads")
	store := checkpoint.DiskHeadStore{Root: root}
	first := testStatement(1)
	if err := store.Initialize(first); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "test-source-seed")
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		t.Fatal(err)
	}
	private := ed25519.NewKeyFromSeed(seed)
	pinned := private.Public().(ed25519.PublicKey)
	if err := os.WriteFile(keyPath, seed, 0600); err != nil {
		t.Fatal(err)
	}
	clear(seed)
	pinHex := hex.EncodeToString(pinned)
	socket := filepath.Join(t.TempDir(), "source.sock")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	child, boot := launchDurableSourceChild(t, ctx, root, socket, keyPath, pinHex)
	if !bytes.Equal(boot.PublicKey, pinned) {
		t.Fatal("first source process returned unexpected signing key")
	}
	provider := checkpoint.UnixHeadProvider{SocketPath: socket, ExpectedUID: uint32(os.Geteuid())}
	trust := checkpoint.Trust{Issuer: first.Issuer, KeyID: first.KeyID, PublicKey: pinned}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, boot.Proof, trust, first.ScopeDigest, first.JournalDigest); err != nil {
		t.Fatalf("first challenge failed with pinned key: %v", err)
	}
	if err := store.Advance(testStatement(2)); err != nil {
		t.Fatal(err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	_ = os.Remove(socket)
	child, restarted := launchDurableSourceChild(t, ctx, root, socket, keyPath, pinHex)
	if restarted.Proof.Statement.Sequence != 2 || !bytes.Equal(restarted.PublicKey, pinned) {
		t.Fatal("durable head or signing identity did not survive fixture restart")
	}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, restarted.Proof, trust, first.ScopeDigest, restarted.Proof.Statement.JournalDigest); err != nil {
		t.Fatalf("restarted source could not prove head under original pinned key: %v", err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	_ = os.Remove(socket)
	otherSeed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(otherSeed); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, otherSeed, 0600); err != nil {
		t.Fatal(err)
	}
	clear(otherSeed)
	_, substituted := launchDurableSourceChild(t, ctx, root, socket, keyPath, pinHex)
	if bytes.Equal(substituted.PublicKey, pinned) {
		t.Fatal("fixture key substitution did not take effect")
	}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, substituted.Proof, trust, first.ScopeDigest, substituted.Proof.Statement.JournalDigest); !errors.Is(err, checkpoint.ErrSignature) {
		t.Fatalf("substituted signing key was accepted by verifier: %v", err)
	}
	if _, err := provider.CurrentHead(ctx, checkpoint.HeadRequest{ScopeDigest: first.ScopeDigest, Nonce: testStatement(2).JournalDigest}); !errors.Is(err, checkpoint.ErrHeadUnavailable) {
		t.Fatalf("source signed with a key different from the pinned fixture key: %v", err)
	}
}

func TestTestOnlySigningFixtureRejectsUnsafeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed")
	if err := os.WriteFile(path, make([]byte, ed25519.SeedSize), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readTestOnlySigner(path); err == nil {
		t.Fatal("world-readable test signer fixture accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "seed-link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readTestOnlySigner(link); err == nil {
		t.Fatal("symlinked signing fixture accepted")
	}
}
