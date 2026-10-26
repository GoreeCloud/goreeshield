//go:build linux

package checkpoint

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const unixHeadMaxResponse = 4096

// UnixHeadProvider connects only to a private Linux Unix socket whose owner
// and SO_PEERCRED UID match a separately configured source service UID.
// It does not provision that identity or any source signing keys.
type UnixHeadProvider struct {
	SocketPath  string
	ExpectedUID uint32
}

type unixHeadRequest struct {
	Operation string   `json:"operation"`
	Scope     [32]byte `json:"scope"`
	Nonce     [32]byte `json:"nonce"`
}

// CurrentHead performs a bounded, private local transport exchange. Returned
// signatures must still be verified through VerifyWithFreshHead.
func (p UnixHeadProvider) CurrentHead(ctx context.Context, request HeadRequest) (HeadProof, error) {
	if ctx == nil || ctx.Err() != nil ||
		!filepath.IsAbs(p.SocketPath) || filepath.Clean(p.SocketPath) != p.SocketPath ||
		emptyDigest(request.ScopeDigest) || emptyDigest(request.Nonce) {
		return HeadProof{}, ErrHeadUnavailable
	}
	info, err := os.Lstat(p.SocketPath)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0077 != 0 {
		return HeadProof{}, ErrHeadUnavailable
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != p.ExpectedUID {
		return HeadProof{}, ErrHeadUnavailable
	}
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(bounded, "unix", p.SocketPath)
	if err != nil {
		return HeadProof{}, ErrHeadUnavailable
	}
	defer conn.Close()
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return HeadProof{}, ErrHeadUnavailable
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return HeadProof{}, ErrHeadUnavailable
	}
	var credentials *syscall.Ucred
	var credentialError error
	if err := raw.Control(func(fd uintptr) {
		credentials, credentialError = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credentialError != nil || credentials == nil || credentials.Uid != p.ExpectedUID {
		return HeadProof{}, ErrHeadUnavailable
	}
	deadline, _ := bounded.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return HeadProof{}, ErrHeadUnavailable
	}
	if err := json.NewEncoder(conn).Encode(unixHeadRequest{Operation: "head", Scope: request.ScopeDigest, Nonce: request.Nonce}); err != nil {
		return HeadProof{}, ErrHeadUnavailable
	}
	decoder := json.NewDecoder(io.LimitReader(conn, unixHeadMaxResponse+1))
	decoder.DisallowUnknownFields()
	var response HeadProof
	if err := decoder.Decode(&response); err != nil {
		return HeadProof{}, ErrHeadUnavailable
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return HeadProof{}, ErrHeadUnavailable
	}
	return response, nil
}
