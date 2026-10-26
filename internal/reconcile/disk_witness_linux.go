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
