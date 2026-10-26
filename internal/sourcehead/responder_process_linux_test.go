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
	Proof     checkpoint.Proof  `json:"proof"`
}
type durableWireRequest struct {
	Operation string   `json:"operation"`
	Scope     [32]byte `json:"scope"`
	Nonce     [32]byte `json:"nonce"`
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
	responder := Responder{Store: store, Signer: private, Issuer: statement.Issuer, KeyID: statement.KeyID, PinnedPublicKey: public}
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
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
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

func launchDurableSourceChild(t *testing.T, ctx context.Context, root, socket string) (*exec.Cmd, durableBoot) {
	t.Helper()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDurableSourceHeadChild$")
	cmd.Env = append(os.Environ(),
		"GS_OWNING_HEAD_CHILD=1", "GS_OWNING_HEAD_ROOT="+root,
		"GS_OWNING_HEAD_SOCKET="+socket)
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	line, err := bufio.NewReader(output).ReadBytes('\n')
	if err != nil {
		t.Fatalf("source child readiness missing: %v", err)
	}
	var boot durableBoot
	if err := json.Unmarshal(line, &boot); err != nil {
		t.Fatal(err)
	}
	return cmd, boot
}

func TestDurableSignerAcrossSourceProcessRestart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "owning-source")
	socket := filepath.Join(t.TempDir(), "source.sock")
	store := checkpoint.DiskHeadStore{Root: root}
	first := testStatement(1)
	if err := store.Initialize(first); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	child, firstBoot := launchDurableSourceChild(t, ctx, root, socket)
	provider := checkpoint.UnixHeadProvider{SocketPath: socket, ExpectedUID: uint32(os.Geteuid())}
	oldTrust := checkpoint.Trust{Issuer: first.Issuer, KeyID: first.KeyID, PublicKey: firstBoot.PublicKey}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, firstBoot.Proof, oldTrust, first.ScopeDigest, first.JournalDigest); err != nil {
		t.Fatalf("first independently signed source head rejected: %v", err)
	}
	if err := store.Advance(testStatement(2)); err != nil {
		t.Fatal(err)
	}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, firstBoot.Proof, oldTrust, first.ScopeDigest, first.JournalDigest); !errors.Is(err, checkpoint.ErrHeadStale) {
		t.Fatalf("old checkpoint accepted after durable advance: %v", err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	if err := os.Remove(socket); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	_, newBoot := launchDurableSourceChild(t, ctx, root, socket)
	if newBoot.Proof.Statement.Sequence != 2 {
		t.Fatalf("head did not survive source process restart: %d", newBoot.Proof.Statement.Sequence)
	}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, firstBoot.Proof, oldTrust, first.ScopeDigest, first.JournalDigest); !errors.Is(err, checkpoint.ErrSignature) {
		t.Fatalf("new signer was silently trusted under old pinned key: %v", err)
	}
	newTrust := checkpoint.Trust{Issuer: first.Issuer, KeyID: first.KeyID, PublicKey: newBoot.PublicKey}
	if err := checkpoint.VerifyWithFreshHead(ctx, provider, newBoot.Proof, newTrust, first.ScopeDigest, first.JournalDigest); err != nil {
		t.Fatalf("fixture current head rejected with freshly pinned test key: %v", err)
	}
}
