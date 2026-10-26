//go:build linux

package checkpoint

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func headScope() [32]byte {
	return sha256.Sum256([]byte("synthetic source scope"))
}

func headStatement(sequence uint64) Statement {
	return Statement{
		Version: Version,
		Issuer: "fixture-owner",
		KeyID: "fixture-key",
		ScopeDigest: headScope(),
		Sequence: sequence,
		JournalDigest: sha256.Sum256([]byte{byte(sequence)}),
	}
}
