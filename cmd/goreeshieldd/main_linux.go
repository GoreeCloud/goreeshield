//go:build linux

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/service"
)

func main() {
	stateFile := flag.String("state", "", "absolute path for local synthetic choices")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "goreeshieldd: unexpected argument")
		os.Exit(2)
	}
	path := *stateFile
	if path == "" {
		var err error
		path, err = choices.DefaultPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "goreeshieldd:", err)
			os.Exit(1)
		}
	}
	server := service.Server{Store: choices.Store{Path: path}}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := server.Run(ctx, nil); err != nil {
		fmt.Fprintln(os.Stderr, "goreeshieldd:", err)
		os.Exit(1)
	}
}
