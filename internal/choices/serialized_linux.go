//go:build linux

package choices

import "errors"

// WithAllowed serializes an in-process synthetic action with local changes to
// the same purpose's choice. The callback is invoked only while that purpose
// has a current, affirmative grant. The local file lock is held throughout
// the callback, so cooperating writers cannot complete withdrawal first.
//
// This is only a LOCAL SYNTHETIC TEST PRIMITIVE. The callback must be fast,
// must not recursively access this Store, and must not perform external I/O.
// It neither authenticates other products nor enforces their data processing.
// False means denied without executing the callback. Errors fail closed.
func (s Store) WithAllowed(purposeID string, action func() error) (bool, error) {
	if _, ok := FindPurpose(purposeID); !ok {
		return false, errors.New("unrecognized synthetic processing purpose")
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
		if !state.Allowed(purposeID) {
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
