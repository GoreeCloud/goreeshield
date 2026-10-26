//go:build linux

package reconcile

import "testing"

func TestMarkerValidationPending(t *testing.T) {
	if journalMarkerContents == "" {
		t.Fatal("empty marker version")
	}
}
