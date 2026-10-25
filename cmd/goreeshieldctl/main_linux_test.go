//go:build linux

package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/service"
)

func TestReadOnlyControlClient(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "private", "choices.json")
	srv := service.Server{Store: choices.Store{Path: statePath}}
	ctx, stop := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx, ready) }()
	select {
	case <-ready:
	case err := <-done:
		stop()
		t.Fatalf("server start: %v", err)
	case <-time.After(3*time.Second):
		stop()
		t.Fatal("server start timeout")
	}
	defer func() {
		stop()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}()
	var out bytes.Buffer
	if err := run([]string{"status", "--state", statePath}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "demo-local-metrics: false") {
		t.Fatalf("expected default deny, got %s", out.String())
	}
	out.Reset()
	if err := run([]string{"notice", "--state", statePath, "--purpose", "demo-local-metrics"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "version 1") {
		t.Fatalf("unexpected notice: %q", out.String())
	}
	if err := srv.Store.Update(func(s *choices.State) error {
		return s.Grant("demo-local-metrics", "1", true, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"check", "--state", statePath, "--purpose", "demo-local-metrics"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "true") {
		t.Fatalf("expected local grant, got %q", out.String())
	}
	if err := run([]string{"grant", "--state", statePath, "--purpose", "demo-local-metrics"}, &out); err == nil {
		t.Fatal("control client should reject writes")
	}
}
