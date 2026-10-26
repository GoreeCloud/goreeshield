//go:build linux

package checkpoint

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type subprocessHeadBoot struct {
	PublicKey ed25519.PublicKey `json:"public_key"`
	Proof Proof `json:"proof"`
	Scope [32]byte `json:"scope"`
	Journal [32]byte `json:"journal"`
}

// TestUnixHeadChild is a native test-only process. Its ephemeral secret key
// never crosses its process boundary; this is not production key custody.
func TestUnixHeadChild(t *testing.T) {
	if os.Getenv("GOREE_SHIELD_HEAD_CHILD") != "1" {
		return
	}
	path := os.Getenv("GOREE_SHIELD_HEAD_SOCKET")
	if path == "" {
		t.Fatal("missing fixture socket")
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	scope := sha256.Sum256([]byte("test-only-source-scope"))
	journal := sha256.Sum256([]byte("test-only-journal"))
	statement := Statement{
		Version: Version, Issuer: "fixture-source", KeyID: "fixture-key",
		ScopeDigest: scope, JournalDigest: journal, Sequence: 23,
	}
	payload, err := SigningPayload(statement)
	if err != nil {
		t.Fatal(err)
	}
	proof := Proof{Statement: statement, Signature: ed25519.Sign(private, payload)}
	socket, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(subprocessHeadBoot{PublicKey: public, Proof: proof, Scope: scope, Journal: journal}); err != nil {
		t.Fatal(err)
	}
	for {
		conn, err := socket.Accept()
		if err != nil {
			t.Fatal(err)
		}
		var request unixHeadRequest
		decoder := json.NewDecoder(conn)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err == nil && request.Operation == "head" {
			head := HeadProof{Statement: statement, Nonce: request.Nonce}
			b, signErr := HeadSigningPayload(head)
			if signErr == nil {
				head.Signature = ed25519.Sign(private, b)
				_ = json.NewEncoder(conn).Encode(head)
			}
		}
		_ = conn.Close()
	}
}

func TestUnixHeadProviderSeparateProcess(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "source.sock")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUnixHeadChild$")
	cmd.Env = append(os.Environ(),
		"GOREE_SHIELD_HEAD_CHILD=1",
		"GOREE_SHIELD_HEAD_SOCKET="+socket,
	)
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
		t.Fatalf("independent child did not become ready: %v", err)
	}
	var boot subprocessHeadBoot
	if err := json.Unmarshal(line, &boot); err != nil {
		t.Fatal(err)
	}
	provider := UnixHeadProvider{SocketPath: socket, ExpectedUID: uint32(os.Geteuid())}
	trust := Trust{Issuer: boot.Proof.Statement.Issuer, KeyID: boot.Proof.Statement.KeyID, PublicKey: boot.PublicKey}
	if err := VerifyWithFreshHead(ctx, provider, boot.Proof, trust, boot.Scope, boot.Journal); err != nil {
		t.Fatalf("independent test process failed signed-head verification: %v", err)
	}
	provider.ExpectedUID++
	if _, err := provider.CurrentHead(ctx, HeadRequest{ScopeDigest: boot.Scope, Nonce: sha256.Sum256([]byte("test request"))}); !errors.Is(err, ErrHeadUnavailable) {
		t.Fatalf("wrong source OS peer UID was trusted: %v", err)
	}
	provider.ExpectedUID--
	if err := os.Chmod(socket, 0666); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.CurrentHead(ctx, HeadRequest{ScopeDigest: boot.Scope, Nonce: sha256.Sum256([]byte("other request"))}); !errors.Is(err, ErrHeadUnavailable) {
		t.Fatalf("public source socket permissions were accepted: %v", err)
	}
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if _, err := provider.CurrentHead(ctx, HeadRequest{ScopeDigest: boot.Scope, Nonce: sha256.Sum256([]byte("offline request"))}); !errors.Is(err, ErrHeadUnavailable) {
		t.Fatalf("dead source process was accepted: %v", err)
	}
}
