//go:build linux

// Package contract defines an internal, untrusted-data-free synthetic adapter
// envelope. This is not a public remote API or a production security token.
package contract

import (
	"errors"
	"fmt"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

const Version = 1

// Request records the synthetic purpose and exact local revision previously
// observed. The caller must recheck at execution time; this is not a grant.
type Request struct {
	Version        int
	OperationID    string
	PurposeID      string
	NoticeVersion  string
	ChoiceRevision uint64
}

// Validate performs structural validation, NOT identity verification.
func (r Request) Validate() error {
	if r.Version != Version {
		return fmt.Errorf("unsupported contract version %d", r.Version)
	}
	if len(r.OperationID) < 3 || len(r.OperationID) > 64 {
		return errors.New("synthetic operation id length out of range")
	}
	for _, c := range r.OperationID {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return errors.New("synthetic operation id must contain only lowercase ASCII letters, digits, hyphen or underscore")
		}
	}
	purpose, known := choices.FindPurpose(r.PurposeID)
	if !known || r.NoticeVersion != purpose.NoticeVersion || r.ChoiceRevision == 0 {
		return errors.New("unknown purpose, outdated notice, or missing revision")
	}
	return nil
}

// Result is an ephemeral observation of a LOCAL synthetic attempt. It is not
// a consent certificate, cryptographically signed receipt or durable record.
// A duplicate never executes an action. PriorStatus is historical only.
type Result struct {
	OperationID string
	Status      string
	Executed    bool
	PriorStatus string
}
