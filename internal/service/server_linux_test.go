//go:build linux

package service

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

func serveTest(t *testing.T) (Server, string, context.CancelFunc, <-chan error) {
	t.Helper()
	state := filepath.Join(t.TempDir(), "private", "choices.json")
	server := Server{Store: choices.Store{Path: state}}
	path, err := server.SocketPath()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, ready) }()
	select {
	case <-ready:
	case err := <-done:
		cancel()
		t.Fatalf("service failed to start: %v", err)
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("service startup timeout")
	}
	return server, path, cancel, done
}

func ask(t *testing.T, socketPath string, req Request) Response {
	t.Helper()
	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		t.Fatal(err)
	}
	var result Response
	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestReadOnlyServiceAndWithdrawal(t *testing.T) {
	server, socketPath, cancel, done := serveTest(t)
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("service failed graceful shutdown")
		}
	}()
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("socket permissions too broad: %s", info.Mode())
	}
	denied := ask(t, socketPath, Request{Action: "check", Purpose: "demo-local-metrics"})
	if !denied.OK || denied.Allowed == nil || *denied.Allowed {
		t.Fatalf("default should deny: %+v", denied)
	}
	notice := ask(t, socketPath, Request{Action: "notice", Purpose: "demo-local-metrics"})
	if !notice.OK || notice.Purpose == nil || notice.Purpose.NoticeVersion != "1" {
		t.Fatalf("bad notice: %+v", notice)
	}
	if result := ask(t, socketPath, Request{Action: "grant", Purpose: "demo-local-metrics"}); result.OK {
		t.Fatal("service must reject any write action")
	}
	if err := server.Store.Update(func(state *choices.State) error {
		return state.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	granted := ask(t, socketPath, Request{Action: "check", Purpose: "demo-local-metrics"})
	if !granted.OK || granted.Allowed == nil || !*granted.Allowed {
		t.Fatalf("service did not reflect grant: %+v", granted)
	}
	if err := server.Store.Update(func(state *choices.State) error {
		return state.Withdraw("demo-local-metrics", time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	revoked := ask(t, socketPath, Request{Action: "check", Purpose: "demo-local-metrics"})
	if !revoked.OK || revoked.Allowed == nil || *revoked.Allowed {
		t.Fatalf("service failed to reflect withdrawal: %+v", revoked)
	}
	status := ask(t, socketPath, Request{Action: "status"})
	if !status.OK || len(status.Statuses) != len(choices.Registry()) {
		t.Fatalf("bad status: %+v", status)
	}
	for _, p := range status.Statuses {
		if p.Allowed {
			t.Fatal("other purposes should not receive grant")
		}
	}
}

func TestServiceRejectsUnknownActionsAndUnsafeSocketPath(t *testing.T) {
	_, socket, cancel, done := serveTest(t)
	other := ask(t, socket, Request{Action: "unknown"})
	if other.OK {
		t.Fatal("unknown command accepted")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	// A stale socket is never removed implicitly. The operator must inspect it.
	seed := filepath.Join(t.TempDir(), "private", "choices.json")
	store := choices.Store{Path: seed}
	if _, err := store.Read(); err != nil {
		t.Fatal(err)
	}
	srv := Server{Store: store}
	path, err := srv.SocketPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("do not overwrite"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := srv.Run(context.Background(), nil); err == nil {
		t.Fatal("existing socket path was overwritten")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "do not overwrite" {
		t.Fatal("existing path was changed", err)
	}
}
