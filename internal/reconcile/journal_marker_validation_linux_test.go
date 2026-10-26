//go:build linux

package reconcile

import (
	"testing"
	"os"
)

func TestMarkerValidationPending(t *testing.T) {
	if journalMarkerContents == "" {
		t.Fatal("empty marker version")
	}
}
