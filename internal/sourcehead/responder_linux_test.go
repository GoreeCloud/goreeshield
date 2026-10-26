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
		ScopeDigest: sha256.Sum256([]byte("synthetic-scope")),
		Sequence: sequence,
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
