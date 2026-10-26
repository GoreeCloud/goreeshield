//go:build linux

package synthetic

import (
	"context"
	"sync"

	"github.com/GoreeCloud/goreeshield/internal/authz"
	"github.com/GoreeCloud/goreeshield/internal/contract"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// ScopedProcessor models one in-memory synthetic engine per verified
// tenant/subject pair. It never connects GoreeCloud Identity or real products.
type ScopedProcessor struct {
	Guard     authz.Guard
	Directory scoped.Directory
	mu        sync.Mutex
	engines   map[string]*Processor
}

// Process requires a matching trusted tenant AND subject claim before resolving
// the local choice file. Caller-supplied identifiers are not authorization.
func (p *ScopedProcessor) Process(
	ctx context.Context, credential, tenant, subject string, request contract.Request,
) (contract.Result, error) {
	result := contract.Result{OperationID: request.OperationID}
	if p == nil {
		return result, authz.ErrDenied
	}
	if err := p.Guard.CheckTenant(ctx, credential, tenant, subject, "privacy.synthetic.execute"); err != nil {
		return result, err
	}
	store, err := p.Directory.Store(tenant, subject)
	if err != nil {
		return result, err
	}
	p.mu.Lock()
	if p.engines == nil {
		p.engines = make(map[string]*Processor)
	}
	engine := p.engines[store.Path]
	if engine == nil {
		engine = &Processor{Store: store, Sink: &Sink{}}
		p.engines[store.Path] = engine
	}
	p.mu.Unlock()
	return engine.Process(request)
}
