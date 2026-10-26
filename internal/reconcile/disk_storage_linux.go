//go:build linux

package reconcile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// diskLedger binds one synthetic counter and its exact operation history in
// the same atomically replaced local file. It is not an audit or legal ledger.
type diskLedger struct {
	Version int               `json:"version"`
	Count   int               `json:"count"`
	Records map[string]record `json:"records"`
}

const maxLedgerBytes = 1024 * 1024

// ErrCommitUncertain means the new source record may already be visible.
// Resolve its exact operation ID through source inspection before retrying.
var ErrCommitUncertain = errors.New("synthetic source journal commit outcome uncertain")

func sourceLedgerPath(choicePath string) string {
	return filepath.Join(filepath.Dir(choicePath), "journal-"+filepath.Base(choicePath))
}

func (s diskLedger) validate() error {
	if s.Version != 1 || s.Count < 0 || len(s.Records) > MaxSourceRecords {
		return errors.New("invalid synthetic journal schema or bounds")
	}
	applied := 0
	for id, rec := range s.Records {
		if id != rec.Request.OperationID || rec.Request.Validate() != nil {
			return errors.New("invalid synthetic journal operation")
		}
		switch rec.Status {
		case "applied":
			applied++
		case "denied":
		default:
			return errors.New("invalid synthetic journal status")
		}
	}
	if applied != s.Count {
		return fmt.Errorf("synthetic counter and source records disagree")
	}
	return nil
}

func ownedPrivateRegular(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 ||
		!ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 {
		return errors.New("synthetic journal file must be owned, private, regular, and have one hard link")
	}
	return nil
}

func readSourceLedger(path string) (diskLedger, error) {
	marked, markErr := journalMarked(path)
	if markErr != nil {
		return diskLedger{}, markErr
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		if marked {
			return diskLedger{}, ErrJournalMissing
		}
		return diskLedger{Version: 1, Records: make(map[string]record)}, nil
	}
	if err != nil {
		return diskLedger{}, err
	}
	defer file.Close()
	if err := ownedPrivateRegular(file); err != nil {
		return diskLedger{}, err
	}
	info, err := file.Stat()
	if err != nil || info.Size() > maxLedgerBytes {
		return diskLedger{}, errors.New("synthetic journal exceeds read limit or cannot be inspected")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxLedgerBytes+1))
	decoder.DisallowUnknownFields()
	var ledger diskLedger
	if err := decoder.Decode(&ledger); err != nil {
		return diskLedger{}, fmt.Errorf("synthetic journal decode failed: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return diskLedger{}, errors.New("synthetic journal contains trailing data")
	}
	if err := ledger.validate(); err != nil {
		return diskLedger{}, err
	}
	if err := verifyJournalWitness(path); err != nil {
		return diskLedger{}, err
	}
	if ledger.Records == nil {
		ledger.Records = make(map[string]record)
	}
	return ledger, nil
}

func writeSourceLedger(path string, ledger diskLedger) error {
	return writeSourceLedgerWithCheckpoint(path, ledger, nil)
}

func writeSourceLedgerWithCheckpoint(path string, ledger diskLedger, checkpoint func(string) error) error {
	if err := ledger.validate(); err != nil {
		return err
	}
	parent := filepath.Dir(path)
	temp, err := os.CreateTemp(parent, ".journal-*.tmp")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if err := json.NewEncoder(temp).Encode(ledger); err != nil {
		temp.Close()
		return err
	}
	if info, err := temp.Stat(); err != nil || info.Size() > maxLedgerBytes {
		temp.Close()
		return errors.New("synthetic journal exceeds write limit")
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := initializeJournalMarker(path); err != nil {
		return err
	}
	if checkpoint != nil {
		if err := checkpoint("before_rename"); err != nil {
			return err
		}
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	if checkpoint != nil {
		if err := checkpoint("after_rename"); err != nil {
			return fmt.Errorf("%w: after rename: %v", ErrCommitUncertain, err)
		}
	}
	dir, err := os.Open(parent)
	if err != nil {
		return fmt.Errorf("%w: parent directory open failed: %v", ErrCommitUncertain, err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("%w: parent directory sync failed: %v", ErrCommitUncertain, err)
	}
	return nil
}

// withSourceLedger serializes cooperating goroutines AND separate processes.
// Lock ordering is journal -> local choice lock; no reverse order is permitted.
func withSourceLedger(path string, fn func(*diskLedger) error) error {
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := ownedPrivateRegular(lock); err != nil {
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	ledger, err := readSourceLedger(path)
	if err != nil {
		return err
	}
	return fn(&ledger)
}
