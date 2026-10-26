//go:build linux

package reconcile

import (
	"context"

	"github.com/GoreeCloud/goreeshield/internal/contract"
	"github.com/GoreeCloud/goreeshield/internal/scoped"
)

// DiskSource persists ONLY a synthetic source counter and operation records.
// It has no external effects, production identity or source integrations.
type DiskSource struct {
	Directory scoped.Directory
}
