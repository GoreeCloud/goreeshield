//go:build linux

package reconcile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var ErrJournalWitness = errors.New("synthetic source journal witness invalid")
var ErrJournalRollback = errors.New("synthetic journal does not match acknowledged witness")

const witnessPrefix = "goreeshield-source-witness-v1:"

func journalWitnessPath(path string) string {
	return path + ".witness"
}

// An absent witness is accepted only for legacy synthetic journal migration.
// This sidecar does not protect against tampering with both same-UID files.
func readJournalWitness(path string) (string, bool, error) {
	f, err := os.OpenFile(journalWitnessPath(path), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	if err := ownedPrivateRegular(f); err != nil {
		return "", false, fmt.Errorf("%w: %v", ErrJournalWitness, err)
	}
	raw, err := io.ReadAll(io.LimitReader(f, 160))
	if err != nil {
		return "", false, err
	}
	value := string(raw)
	if !strings.HasPrefix(value, witnessPrefix) || !strings.HasSuffix(value, "\n") {
		return "", false, ErrJournalWitness
	}
	hash := strings.TrimSuffix(strings.TrimPrefix(value, witnessPrefix), "\n")
	if len(hash) != sha256.Size*2 {
		return "", false, ErrJournalWitness
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", false, ErrJournalWitness
	}
	return hash, true, nil
}

// Checksum the exact published JSON bytes, not a reconstructed map; source
// operations must already be serialized by the cooperating journal lock.
func hashCurrentJournal(path string) (string, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := ownedPrivateRegular(f); err != nil {
		return "", err
	}
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > maxLedgerBytes {
		return "", ErrJournalWitness
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxLedgerBytes+1))
	if err != nil {
		return "", err
	}
	if n > maxLedgerBytes {
		return "", ErrJournalWitness
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyJournalWitness(path string) error {
	want, found, err := readJournalWitness(path)
	if err != nil || !found {
		return err
	}
	got, err := hashCurrentJournal(path)
	if err != nil {
		return err
	}
	if want != got {
		return ErrJournalRollback
	}
	return nil
}
