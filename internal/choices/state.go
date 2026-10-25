package choices

import (
	"errors"
	"fmt"
	"time"
)

// Choice stores only a local user's last explicit decision. It is NOT an
// authentication token, a regulatory consent certificate, or a durable audit log.
type Choice struct {
	Granted       bool      `json:"granted"`
	NoticeVersion string    `json:"notice_version"`
	ChangedAt     time.Time `json:"changed_at"`
}

type State struct {
	SchemaVersion int               `json:"schema_version"`
	Choices       map[string]Choice `json:"choices"`
}

func EmptyState() State { return State{SchemaVersion: 1, Choices: map[string]Choice{}} }

// Allowed always fails closed for missing, unknown, or outdated grants.
func (s State) Allowed(purposeID string) bool {
	purpose, ok := FindPurpose(purposeID)
	if !ok {
		return false
	}
	choice, ok := s.Choices[purposeID]
	return ok && choice.Granted && choice.NoticeVersion == purpose.NoticeVersion && !choice.ChangedAt.IsZero()
}

func (s *State) Grant(purposeID, noticeVersion string, affirmative bool, now time.Time) error {
	purpose, ok := FindPurpose(purposeID)
	if !ok {
		return errors.New("unknown processing purpose")
	}
	if !affirmative {
		return errors.New("grant requires explicit acknowledgement of the current notice")
	}
	if noticeVersion != purpose.NoticeVersion {
		return fmt.Errorf("notice version mismatch for %q", purposeID)
	}
	if now.IsZero() {
		return errors.New("decision time is required")
	}
	if s.Choices == nil {
		s.Choices = map[string]Choice{}
	}
	s.Choices[purposeID] = Choice{Granted: true, NoticeVersion: purpose.NoticeVersion, ChangedAt: now.UTC()}
	return nil
}

func (s *State) Withdraw(purposeID string, now time.Time) error {
	purpose, ok := FindPurpose(purposeID)
	if !ok {
		return errors.New("unknown processing purpose")
	}
	if now.IsZero() {
		return errors.New("decision time is required")
	}
	if s.Choices == nil {
		s.Choices = map[string]Choice{}
	}
	s.Choices[purposeID] = Choice{Granted: false, NoticeVersion: purpose.NoticeVersion, ChangedAt: now.UTC()}
	return nil
}

func (s State) Validate() error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("unsupported state schema version %d", s.SchemaVersion)
	}
	for id, c := range s.Choices {
		if _, ok := FindPurpose(id); !ok {
			return fmt.Errorf("unrecognized purpose in state: %q", id)
		}
		if c.NoticeVersion == "" || c.ChangedAt.IsZero() {
			return fmt.Errorf("invalid stored decision for %q", id)
		}
	}
	return nil
}
