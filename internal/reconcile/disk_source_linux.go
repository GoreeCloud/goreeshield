//go:build linux

package reconcile

import (
	"context"

	"github.com/GoreeCloud/goreeshield/internal/contract"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// DiskSource persists ONLY a synthetic source counter and operation records.
// It has no external effects, production identity or source integrations.
type DiskSource struct {
	Directory scoped.Directory
}

// Apply records the exact local synthetic effect and operation ID together.
// The journal lock excludes cooperating processes; the choice lock prevents
// withdrawal from overtaking an in-progress synthetic commit.
func (s *DiskSource) Apply(ctx context.Context, tenant, subject string, req contract.Request) error {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return ErrUnavailable
	}
	if err := req.Validate(); err != nil {
		return err
	}
	store, err := s.Directory.Store(tenant, subject)
	if err != nil {
		return err
	}
	path := sourceLedgerPath(store.Path)
	return withSourceLedger(path, func(j *diskLedger) error {
		if previous, exists := j.Records[req.OperationID]; exists {
			if previous.Request != req {
				return ErrConflict
			}
			return nil
		}
		if len(j.Records) >= MaxSourceRecords {
			return ErrFull
		}
		if ctx.Err() != nil {
			return ErrUnavailable
		}
		performed, err := store.WithMatchingRevision(req.PurposeID, req.NoticeVersion, req.ChoiceRevision, func() error {
			if ctx.Err() != nil {
				return ErrUnavailable
			}
			j.Count++
			j.Records[req.OperationID] = record{Request: req, Status: "applied"}
			return writeSourceLedger(path, *j)
		})
		if err != nil {
			return err
		}
		if !performed {
			j.Records[req.OperationID] = record{Request: req, Status: "denied"}
			return writeSourceLedger(path, *j)
		}
		return nil
	})
}
