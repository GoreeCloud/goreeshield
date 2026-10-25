//go:build linux

// Package synthetic models an OPTIONAL downstream processor. It intentionally
// has no network transport, persistent event store or personal-data input.
package synthetic

import (
	"errors"
	"sync"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

// Sink counts synthetic events in memory only. These counts are test
// observations, not user data, privacy receipts, or operational metrics.
type Sink struct {
	mu     sync.Mutex
	counts map[string]int
}

func (s *Sink) record(purpose string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts == nil {
		s.counts = make(map[string]int)
	}
	s.counts[purpose]++
}

func (s *Sink) Count(purpose string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[purpose]
}

// Adapter is a fixture for checking that an optional source operation actually
// depends on the CURRENT choice. No real source system calls this code.
type Adapter struct {
	Store choices.Store
	Sink  *Sink
}

type Outcome struct {
	Purpose  string
	Executed bool
}

// Attempt atomically checks the local choice and increments a synthetic,
// in-memory count only when the choice is affirmative. No data is collected.
func (a Adapter) Attempt(purposeID string) (Outcome, error) {
	out := Outcome{Purpose: purposeID}
	if a.Sink == nil {
		return out, errors.New("synthetic observation sink is required")
	}
	// The action does not call other stores, execute arbitrary code or do IO.
	executed, err := a.Store.WithAllowed(purposeID, func() error {
		a.Sink.record(purposeID)
		return nil
	})
	if err != nil {
		return out, err
	}
	out.Executed = executed
	return out, nil
}
