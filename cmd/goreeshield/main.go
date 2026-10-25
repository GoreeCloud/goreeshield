//go:build linux

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
)

func usage(w io.Writer) {
	fmt.Fprintln(w, "GoreeShield local-only privacy choices prototype (Linux)")
	fmt.Fprintln(w, "Usage: goreeshield <notice|status|check|grant|withdraw> [flags]")
	fmt.Fprintln(w, "  notice   --purpose ID [--state ABSOLUTE_PATH]")
	fmt.Fprintln(w, "  status   [--state ABSOLUTE_PATH]")
	fmt.Fprintln(w, "  check    --purpose ID [--state ABSOLUTE_PATH]")
	fmt.Fprintln(w, "  grant    --purpose ID --notice-version VERSION --acknowledge [--state ABSOLUTE_PATH]")
	fmt.Fprintln(w, "  withdraw --purpose ID [--state ABSOLUTE_PATH]")
	fmt.Fprintln(w, "Only synthetic example purposes are registered. No live data processing is enabled.")
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 { usage(out); return errors.New("command required") }
	cmd := args[0]
	if cmd == "help" || cmd == "--help" { usage(out); return nil }
	switch cmd {
	case "notice", "status", "check", "grant", "withdraw":
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	statePath := fs.String("state", "", "absolute path for private local state")
	purposeID := fs.String("purpose", "", "registered processing purpose")
	noticeVersion := fs.String("notice-version", "", "version of notice reviewed")
	ack := fs.Bool("acknowledge", false, "I reviewed the notice and explicitly choose to enable this example")
	if err := fs.Parse(args[1:]); err != nil { return err }
	if fs.NArg() != 0 { return errors.New("unexpected positional arguments") }
	if cmd != "grant" && (*ack || *noticeVersion != "") { return errors.New("acknowledgement and notice version only apply to grant") }
	if cmd == "status" && *purposeID != "" { return errors.New("status does not accept a purpose") }
	if cmd != "status" && *purposeID == "" { return errors.New("--purpose is required") }
	if cmd != "grant" && *ack { return errors.New("--acknowledge is only for grant") }
	path := *statePath
	if path == "" {
		var err error
		path, err = choices.DefaultPath()
		if err != nil { return err }
	}
	store := choices.Store{Path: path}
	if cmd == "notice" {
		p, ok := choices.FindPurpose(*purposeID)
		if !ok { return errors.New("unknown processing purpose") }
		fmt.Fprintf(out, "%s (notice version %s)\nPurpose: %s\nData: %s\nRecipients: %s\nRetention: %s\n", p.Name, p.NoticeVersion, p.Notice, p.Data, p.Recipient, p.Retention)
		fmt.Fprintln(out, "This is a nonfunctional synthetic demonstration. The optional choice is disabled until explicitly granted.")
		return nil
	}
	if cmd == "grant" {
		if err := store.Update(func(s *choices.State) error {
			return s.Grant(*purposeID, *noticeVersion, *ack, time.Now().UTC())
		}); err != nil { return err }
		fmt.Fprintf(out, "%s: local opt-in recorded; no data processing was started.\n", *purposeID)
		return nil
	}
	if cmd == "withdraw" {
		if err := store.Update(func(s *choices.State) error {
			return s.Withdraw(*purposeID, time.Now().UTC())
		}); err != nil { return err }
		fmt.Fprintf(out, "%s: local choice withdrawn; no external service was contacted.\n", *purposeID)
		return nil
	}
	state, err := store.Read()
	if err != nil { return err }
	if cmd == "check" {
		if _, ok := choices.FindPurpose(*purposeID); !ok { return errors.New("unknown processing purpose") }
		if state.Allowed(*purposeID) { fmt.Fprintln(out, "GRANTED (local demonstration only)") } else { fmt.Fprintln(out, "DENIED (default or withdrawn)") }
		return nil
	}
	for _, p := range choices.Registry() {
		result := "disabled"
		if state.Allowed(p.ID) { result = "granted for local demonstration" }
		fmt.Fprintf(out, "%s: %s\n", p.ID, result)
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "goreeshield: %v\n", err)
		os.Exit(1)
	}
}
