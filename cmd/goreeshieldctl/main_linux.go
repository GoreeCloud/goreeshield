//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/service"
)

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("expected status, notice, or check")
	}
	action := args[0]
	if action != "status" && action != "notice" && action != "check" {
		return errors.New("only status, notice and check are supported")
	}
	fs := flag.NewFlagSet(action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	statePath := fs.String("state", "", "absolute path of local choice file")
	purpose := fs.String("purpose", "", "synthetic registered purpose")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional argument")
	}
	if (action == "status") != (*purpose == "") {
		return errors.New("provide --purpose for notice/check, not status")
	}
	path := *statePath
	if path == "" {
		var err error
		path, err = choices.DefaultPath()
		if err != nil {
			return err
		}
	}
	socket, err := (service.Server{Store: choices.Store{Path: path}}).SocketPath()
	if err != nil {
		return err
	}
	conn, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("local service unavailable: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	if err := json.NewEncoder(conn).Encode(service.Request{Action: action, Purpose: *purpose}); err != nil {
		return err
	}
	var response service.Response
	if err := json.NewDecoder(io.LimitReader(conn, 64*1024)).Decode(&response); err != nil {
		return fmt.Errorf("invalid local service response: %w", err)
	}
	if !response.OK {
		return fmt.Errorf("local service rejected request: %s", response.Error)
	}
	switch action {
	case "check":
		if response.Allowed == nil {
			return errors.New("missing choice result")
		}
		fmt.Fprintf(out, "%s: %t (synthetic local decision only)\n", *purpose, *response.Allowed)
	case "notice":
		if response.Purpose == nil {
			return errors.New("missing purpose notice")
		}
		p := response.Purpose
		fmt.Fprintf(out, "%s (version %s)\n%s\nData: %s\nRecipient: %s\nRetention: %s\n", p.Name, p.NoticeVersion, p.Notice, p.Data, p.Recipient, p.Retention)
	case "status":
		for _, p := range response.Statuses {
			fmt.Fprintf(out, "%s: %t\n", p.ID, p.Allowed)
		}
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "goreeshieldctl:", err)
		os.Exit(1)
	}
}
