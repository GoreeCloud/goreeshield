//go:build linux

package synthetic

import (
	"context"

	"github.com/GoreeCloud/goreeshield/internal/authz"
	"github.com/GoreeCloud/goreeshield/internal/contract"
)

// GuardedProcessor is a synthetic authorization-boundary test harness.
// No actual GoreeCloud Identity verifier is configured or shipped. It must
// not be exposed as a remote or multi-user production processing endpoint.
type GuardedProcessor struct {
	Guard        authz.Guard
	Engine       *Processor
	LocalSubject string
}

// Process first requires a verified claim for this one synthetic local subject
// and an exact execution scope. Without a verifier it always denies. Subject
// strings do not authorize processing by themselves.
func (p GuardedProcessor) Process(
	ctx context.Context, credential, subject string, request contract.Request,
) (contract.Result, error) {
	result := contract.Result{OperationID: request.OperationID}
	if p.Engine == nil || p.LocalSubject == "" || subject != p.LocalSubject {
		return result, authz.ErrDenied
	}
	if err := p.Guard.Check(ctx, credential, subject, "privacy.synthetic.execute"); err != nil {
		return result, err
	}
	return p.Engine.Process(request)
}
