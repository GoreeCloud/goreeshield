//go:build linux

package sourcehead

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/checkpoint"
)

type durableBoot struct {
	PublicKey ed25519.PublicKey `json:"public_key"`
	Proof checkpoint.Proof `json:"proof"`
}
type durableWireRequest struct {
	Operation string `json:"operation"`
	Scope [32]byte `json:"scope"`
	Nonce [32]byte `json:"nonce"`
}
