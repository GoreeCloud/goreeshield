//go:build linux

package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalCLIFlow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "choices.json")
	call := func(args ...string) (string, error) {
		var out bytes.Buffer
		returnValue := append(args, "--state", path)
		err := run(returnValue, &out)
		return out.String(), err
	}
	if out, err := call("check", "--purpose", "demo-local-metrics"); err != nil || !strings.Contains(out, "DENIED") { t.Fatalf("default: %q %v", out, err) }
	if _, err := call("grant", "--purpose", "demo-local-metrics", "--notice-version", "1"); err == nil { t.Fatal("grant missing acknowledgement") }
	if out, err := call("notice", "--purpose", "demo-local-metrics"); err != nil || !strings.Contains(out, "notice version 1") { t.Fatalf("notice: %q %v", out, err) }
	if _, err := call("grant", "--purpose", "demo-local-metrics", "--notice-version", "1", "--acknowledge"); err != nil { t.Fatal(err) }
	if out, err := call("check", "--purpose", "demo-local-metrics"); err != nil || !strings.Contains(out, "GRANTED") { t.Fatalf("grant: %q %v", out, err) }
	if _, err := call("withdraw", "--purpose", "demo-local-metrics"); err != nil { t.Fatal(err) }
	if out, err := call("check", "--purpose", "demo-local-metrics"); err != nil || !strings.Contains(out, "DENIED") { t.Fatalf("withdraw: %q %v", out, err) }
}
