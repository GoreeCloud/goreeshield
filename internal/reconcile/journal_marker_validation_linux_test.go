//go:build linux

package reconcile

import (
	"testing"
	"os"
	"errors"
	"context"
	"path/filepath"
)

func TestMarkerValidationPending(t *testing.T) {
	if journalMarkerContents == "" {
		t.Fatal("empty marker version")
	}
}
