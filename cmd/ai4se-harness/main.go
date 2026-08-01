package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output, errors io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ai4se-harness serve|run|credentials|demo")
	}
	switch args[0] {
	case "serve":
		return serve(ctx, args[1:], output)
	case "run":
		return runCommand(args[1:], output)
	case "credentials":
		return credentialsCommand(ctx, args[1:], output, errors)
	case "demo":
		return demoCommand(ctx, args[1:], output)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
