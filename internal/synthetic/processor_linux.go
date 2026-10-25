//go:build linux

package synthetic

import (
	"errors"
	"sync"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/contract"
)

// Processor models an in-memory adapter with bounded deduplication. It does
// not persist its operation records across a process restart, make network
// calls, or accept personal data. The caller must NOT use its records as
// production evidence or downstream consent receipts.
type Processor struct {
	Store   choices.Store
	Sink    *Sink
	mu      sync.Mutex
	history map[string]processed
}

type processed struct {
	request contract.Request
	status  string
}

const MaxSyntheticOperations = 1024

// Process serializes duplicate operation IDs and checks the expected consent
// revision under the local choice file lock before executing a synthetic
// in-memory counter increment. It never trusts a previous allowed result.
func (p *Processor) Process(req contract.Request) (contract.Result, error) {
	result := contract.Result{OperationID: req.OperationID}
	if err := req.Validate(); err != nil {
		return result, err
	}
	if p == nil || p.Sink == nil {
		return result, errors.New("synthetic processor requires an in-memory sink")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if previous, exists := p.history[req.OperationID]; exists {
		if previous.request != req {
			result.Status = "conflict"
			return result, nil
		}
		result.Status = "duplicate"
		result.PriorStatus = previous.status
		return result, nil
	}
	if len(p.history) >= MaxSyntheticOperations {
		return result, errors.New("synthetic idempotency table full; refusing to execute")
	}

	performed, err := p.Store.WithMatchingRevision(req.PurposeID, req.NoticeVersion, req.ChoiceRevision, func() error {
		p.Sink.record(req.PurposeID)
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Status = "denied"
	if performed {
		result.Status = "applied"
		result.Executed = true
	}
	if p.history == nil {
		p.history = make(map[string]processed)
	}
	p.history[req.OperationID] = processed{request: req, status: result.Status}
	return result, nil
}
