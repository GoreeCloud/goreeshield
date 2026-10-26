//go:build linux

package reconcile

import (
	"context"
	"encoding/hex"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// VerifySyntheticJournalCheckpoint compares the local synthetic journal with
// a separately signed source statement. Trust and trustedMinimum MUST be
// provisioned by an independent authoritative source, never local sidecars.
// This helper does not connect to, or act for, a real product.
func VerifySyntheticJournalCheckpoint(
	ctx context.Context, directory scoped.Directory, tenant, subject string,
	proof checkpoint.Proof, trust checkpoint.Trust, trustedMinimum uint64,
) error {
	if ctx == nil || ctx.Err() != nil {
		return ErrUnavailable
	}
	store, err := directory.Store(tenant, subject)
	if err != nil {
		return err
	}
	scopeDigest, err := checkpoint.ScopeDigest(tenant, subject)
	if err != nil {
		return err
	}
	path := sourceLedgerPath(store.Path)
	return withSourceLedger(path, func(_ *diskLedger) error {
		if err := ctx.Err(); err != nil {
			return ErrUnavailable
		}
		currentHex, err := hashCurrentJournal(path)
		if err != nil {
			return err
		}
		raw, err := hex.DecodeString(currentHex)
		if err != nil {
			return err
		}
		var current [32]byte
		copy(current[:], raw)
		return checkpoint.Verify(proof, trust, checkpoint.Expected{
			ScopeDigest:     scopeDigest,
			JournalDigest:   current,
			MinimumSequence: trustedMinimum,
		})
	})
}
