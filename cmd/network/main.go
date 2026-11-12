package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kizio806/minecraft-network/internal/networkcli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := (networkcli.App{Out: os.Stdout, Err: os.Stderr}).Execute(ctx, os.Args[1:])
	if err == nil {
		return
	}
	var exit *networkcli.ExitError
	code := 1
	if errors.As(err, &exit) {
		code = exit.Code
	}
	fmt.Fprintln(os.Stderr, "network:", err)
	os.Exit(code)
}
