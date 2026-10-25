//go:build linux

// Package service provides a read-only, same-user Unix domain socket boundary
// for GoreeShield's synthetic Linux privacy choices prototype.
package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

const maxRequestBytes = 1024

type Request struct {
	Action  string `json:"action"`
	Purpose string `json:"purpose,omitempty"`
}

type PurposeStatus struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Allowed       bool   `json:"allowed"`
	NoticeVersion string `json:"notice_version"`
}

type Response struct {
	OK       bool            `json:"ok"`
	Error    string          `json:"error,omitempty"`
	Statuses []PurposeStatus `json:"statuses,omitempty"`
	Purpose  *choices.Purpose `json:"purpose,omitempty"`
	Allowed  *bool           `json:"allowed,omitempty"`
}

type Server struct {
	Store choices.Store
}

// SocketPath confines socket access to the same private directory as choices.
func (s Server) SocketPath() (string, error) {
	if s.Store.Path == "" || !filepath.IsAbs(s.Store.Path) {
		return "", errors.New("absolute local state path required")
	}
	return filepath.Join(filepath.Dir(filepath.Clean(s.Store.Path)), "goreeshield.sock"), nil
}

// Run starts a native Linux Unix socket listener. It deliberately exposes no
// grant, withdrawal, remote API or real application processing operation.
func (s Server) Run(ctx context.Context, ready chan<- struct{}) error {
	// Read initializes and checks the existing private directory and lock.
	if _, err := s.Store.Read(); err != nil {
		return err
	}
	socketPath, err := s.SocketPath()
	if err != nil {
		return err
	}
	if _, err := os.Lstat(socketPath); err == nil {
		return errors.New("socket path already exists: refuse to replace or unlink it")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return err
	}
	defer listener.Close()
	listener.SetUnlinkOnClose(true)
	if err := os.Chmod(socketPath, 0600); err != nil {
		return err
	}
	if ready != nil {
		close(ready)
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			listener.Close()
		case <-done:
		}
	}()
	defer close(done)

	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("unix socket accept: %w", err)
		}
		// Serialize requests and bound individual connection time.
		s.handle(conn)
	}
}

func sameUser(conn *net.UnixConn) (bool, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return false, err
	}
	var credential *syscall.Ucred
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		credential, sockErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return false, err
	}
	if sockErr != nil {
		return false, sockErr
	}
	return credential != nil && credential.Uid == uint32(os.Geteuid()), nil
}

func (s Server) handle(conn *net.UnixConn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	allowed, err := sameUser(conn)
	if err != nil || !allowed {
		_ = json.NewEncoder(conn).Encode(Response{Error: "unauthorized peer"})
		return
	}

	// The protocol is one newline-terminated JSON request per connection.
	// Do not block waiting for the client to close its write side before reply.
	line, err := bufio.NewReader(io.LimitReader(conn, maxRequestBytes+1)).ReadBytes('\n')
	if err != nil || len(line) > maxRequestBytes {
		_ = json.NewEncoder(conn).Encode(Response{Error: "invalid or oversized request"})
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var req Request
	if err := decoder.Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(Response{Error: "invalid request"})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		_ = json.NewEncoder(conn).Encode(Response{Error: "trailing request data"})
		return
	}
	if req.Action == "" || len(req.Purpose) > 128 {
		_ = json.NewEncoder(conn).Encode(Response{Error: "invalid action or purpose"})
		return
	}
	_ = json.NewEncoder(conn).Encode(s.dispatch(req))
}

func (s Server) dispatch(req Request) Response {
	switch req.Action {
	case "notice":
		purpose, ok := choices.FindPurpose(req.Purpose)
		if !ok {
			return Response{Error: "unknown processing purpose"}
		}
		return Response{OK: true, Purpose: &purpose}
	case "status", "check":
		if req.Action == "status" && req.Purpose != "" {
			return Response{Error: "status does not take a purpose"}
		}
		if req.Action == "check" {
			if _, ok := choices.FindPurpose(req.Purpose); !ok {
				return Response{Error: "unknown processing purpose"}
			}
		}
		state, err := s.Store.Read()
		if err != nil {
			return Response{Error: "local state unavailable"}
		}
		if req.Action == "check" {
			allowed := state.Allowed(req.Purpose)
			return Response{OK: true, Allowed: &allowed}
		}
		response := Response{OK: true, Statuses: make([]PurposeStatus, 0)}
		for _, p := range choices.Registry() {
			response.Statuses = append(response.Statuses, PurposeStatus{
				ID: p.ID, Name: p.Name, Allowed: state.Allowed(p.ID), NoticeVersion: p.NoticeVersion,
			})
		}
		return response
	default:
		return Response{Error: "unknown or unsupported action"}
	}
}
