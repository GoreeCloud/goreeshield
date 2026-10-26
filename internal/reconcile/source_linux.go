//go:build linux

// Package reconcile provides an in-memory, synthetic source-observation
// experiment. It does not process personal information or provide production
// evidence, rights fulfillment, durable records, or external integrations.
package reconcile

import (
	"context"
	"errors"
	"sync"

	"github.com/GoreeCloud/goreeshield/internal/contract"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

var ErrConflict = errors.New("synthetic source operation ID reused with different request")
var ErrFull = errors.New("synthetic source operation history full")
var ErrUnavailable = errors.New("synthetic source unavailable or canceled")

const MaxSourceRecords = 1024

// Observation is a local fixture's account of its own action, not an
// independently attested, cryptographically signed or durable receipt.
type Observation struct {
	Found       bool
	Status      string
	OperationID string
	Request     contract.Request
}

type record struct {
	Request contract.Request
	Status  string
}

// MockSource owns a separate in-memory operation journal from the coordinator.
// Both structures are confined to this process; no genuine external product
// has been contacted. It performs no network or filesystem output itself.
type MockSource struct {
	Directory scoped.Directory
	mu        sync.Mutex
	records   map[string]map[string]record
	counts    map[string]int
}

func (s *MockSource) Apply(ctx context.Context, tenant, subject string, req contract.Request) error {
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
	key := store.Path

	// Serialize attempts and duplicate IDs with the source-owned observation.
	// The choice-store lock still protects the actual short synthetic action.
	s.mu.Lock()
	defer s.mu.Unlock()

	history := s.records[key]
	if previous, exists := history[req.OperationID]; exists {
		if previous.Request != req {
			return ErrConflict
		}
		return nil // historical retry, never a second action
	}
	if len(history) >= MaxSourceRecords {
		return ErrFull
	}
	if ctx.Err() != nil {
		return ErrUnavailable
	}

	performed, err := store.WithMatchingRevision(req.PurposeID, req.NoticeVersion, req.ChoiceRevision, func() error {
		if ctx.Err() != nil {
			return ErrUnavailable
		}
		// A synthetic counter is the ONLY source action.
		if s.counts == nil {
			s.counts = make(map[string]int)
		}
		s.counts[key]++
		return nil
	})
	if err != nil {
		return err
	}
	status := "denied"
	if performed {
		status = "applied"
	}
	if s.records == nil {
		s.records = make(map[string]map[string]record)
	}
	if history == nil {
		history = make(map[string]record)
		s.records[key] = history
	}
	history[req.OperationID] = record{Request: req, Status: status}
	return nil
}

// Inspect reads the mock source's operation journal, not an earlier
// coordinator response. Missing observations mean unknown, not denied.
func (s *MockSource) Inspect(ctx context.Context, tenant, subject, operationID string) (Observation, error) {
	var result Observation
	if s == nil || ctx == nil || ctx.Err() != nil {
		return result, ErrUnavailable
	}
	store, err := s.Directory.Store(tenant, subject)
	if err != nil {
		return result, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.records[store.Path][operationID]; ok {
		result = Observation{Found: true, Status: item.Status, OperationID: operationID, Request: item.Request}
	}
	return result, nil
}

// Count returns only the mock source's ephemeral synthetic counter.
func (s *MockSource) Count(tenant, subject string) (int, error) {
	store, err := s.Directory.Store(tenant, subject)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[store.Path], nil
}
