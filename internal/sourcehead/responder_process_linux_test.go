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

// The fixture keeps an ephemeral private key in the child process only.
// On restart a NEW key is generated; the parent must refuse old pinned trust.
func TestDurableSourceHeadChild(t *testing.T) {
	if os.Getenv("GS_OWNING_HEAD_CHILD") != "1" {
		return
	}
	store := checkpoint.DiskHeadStore{Root: os.Getenv("GS_OWNING_HEAD_ROOT")}
	statement, err := store.Read(testStatement(1).ScopeDigest)
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	responder := Responder{Store: store, Signer: private, Issuer: statement.Issuer, KeyID: statement.KeyID}
	socket, err := net.Listen("unix", os.Getenv("GS_OWNING_HEAD_SOCKET"))
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	if err := os.Chmod(os.Getenv("GS_OWNING_HEAD_SOCKET"), 0600); err != nil {
		t.Fatal(err)
	}
	proof := testProof(t, statement, private)
	if err := json.NewEncoder(os.Stdout).Encode(durableBoot{PublicKey: public, Proof: proof}); err != nil {
		t.Fatal(err)
	}
	for {
		conn, err := socket.Accept()
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.SetDeadline(time.Now().Add(2*time.Second))
		var request durableWireRequest
		if err := json.NewDecoder(conn).Decode(&request); err == nil && request.Operation == "head" {
			head, replyErr := responder.Respond(context.Background(), checkpoint.HeadRequest{
				ScopeDigest: request.Scope, Nonce: request.Nonce,
			})
			if replyErr == nil {
				_ = json.NewEncoder(conn).Encode(head)
			}
		}
		_ = conn.Close()
	}
}
