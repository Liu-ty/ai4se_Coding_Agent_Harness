package main

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func runCommand(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repo := flags.String("repo", "", "repository path")
	task := flags.String("task", "", "task text")
	config := flags.String("config", ".ai4se-harness.toml", "configuration file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *task == "" {
		return fmt.Errorf("run requires --repo and --task")
	}
	runtime, err := newLocalRuntime(context.Background(), *repo, localRuntimeOptions{})
	if err != nil {
		return fmt.Errorf("local composition: %w", err)
	}
	defer runtime.Close()
	run, err := runtime.Application.CreateRun(context.Background(), localRunRequest(*repo, *task, *config))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "run created: %s\n", run.ID)
	return err
}
