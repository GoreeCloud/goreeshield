//go:build linux

package reconcile

import (
	"context"
	"crypto/subtle"
	"encoding/hex"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// VerifySyntheticJournalWithFreshHead verifies the current local synthetic
// journal against a source-owned, nonce-bound head response. The provider is
// called OUTSIDE the local journal lock to avoid remote-call deadlocks.
// The journal is checked again afterward to detect a concurrent local write.
// This is point-in-time verification, not a permanent transaction lock.
func VerifySyntheticJournalWithFreshHead(
	ctx context.Context, directory scoped.Directory, tenant, subject string,
	proof checkpoint.Proof, trust checkpoint.Trust, provider checkpoint.HeadProvider,
) error {
	if ctx == nil || ctx.Err() != nil {
		return ErrUnavailable
	}
	store, err := directory.Store(tenant, subject)
	if err != nil {
		return err
	}
	scope, err := checkpoint.ScopeDigest(tenant, subject)
	if err != nil {
		return err
	}
	path := sourceLedgerPath(store.Path)
	readDigest := func() ([32]byte, error) {
		var digest [32]byte
		err := withSourceLedger(path, func(_ *diskLedger) error {
			if ctx.Err() != nil {
				return ErrUnavailable
			}
			hash, err := hashCurrentJournal(path)
			if err != nil {
				return err
			}
			raw, err := hex.DecodeString(hash)
			if err != nil {
				return err
			}
			copy(digest[:], raw)
			return nil
		})
		return digest, err
	}
	first, err := readDigest()
	if err != nil {
		return err
	}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, proof, trust, scope, first); err != nil {
		return err
	}
	second, err := readDigest()
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(first[:], second[:]) != 1 {
		return checkpoint.ErrMismatch
	}
	return nil
}
