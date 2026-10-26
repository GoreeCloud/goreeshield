//go:build linux

package reconcile

import (
	"context"

	"github.com/GoreeCloud/goreeshield/internal/authz"
	"github.com/GoreeCloud/goreeshield/internal/contract"
)

// Source models separate synthetic execution and subsequent observation.
type Source interface {
	Apply(context.Context, string, string, contract.Request) error
	Inspect(context.Context, string, string, string) (Observation, error)
}

type Coordinator struct {
	Guard  authz.Guard
	Source Source
}

type Receipt struct {
	OperationID string
	Status      string
}

// Dispatch requires the synthetic execute scope. Successful submission never
// establishes confirmation; it stays pending until a separate source read.
func (c Coordinator) Dispatch(ctx context.Context, credential, tenant, subject string, req contract.Request) (Receipt, error) {
	out := Receipt{OperationID: req.OperationID, Status: "unresolved"}
	if err := c.Guard.CheckTenant(ctx, credential, tenant, subject, "privacy.synthetic.execute"); err != nil {
		return out, err
	}
	if err := req.Validate(); err != nil {
		return out, err
	}
	if c.Source == nil {
		return out, ErrUnavailable
	}
	if err := c.Source.Apply(ctx, tenant, subject, req); err != nil {
		return out, err
	}
	out.Status = "pending_source_readback"
	return out, nil
}
