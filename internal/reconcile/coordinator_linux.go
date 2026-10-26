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
