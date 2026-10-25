//go:build linux

package choices

import "errors"

// WithMatchingRevision is a LOCAL, short-lived synthetic operation gate.
// It requires an exact notice and choice revision while the same file lock
// excludes cooperating local choice writers. No callback may perform network
// operations or re-enter this Store. A mismatched revision is denied, never
// interpreted as a cached grant or authorization for a remote adapter.
func (s Store) WithMatchingRevision(purposeID, noticeVersion string, revision uint64, action func() error) (bool, error) {
	purpose, known := FindPurpose(purposeID)
	if !known || noticeVersion != purpose.NoticeVersion || revision == 0 {
		return false, errors.New("invalid synthetic purpose, notice, or revision")
	}
	if action == nil {
		return false, errors.New("synthetic action is required")
	}
	performed := false
	err := s.withLock(func(_ string) error {
		state, err := s.readUnlocked()
		if err != nil {
			return err
		}
		if state.Revision != revision || !state.Allowed(purposeID) {
			return nil
		}
		if err := action(); err != nil {
			return err
		}
		performed = true
		return nil
	})
	return performed, err
}
