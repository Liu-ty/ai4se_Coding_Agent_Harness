package main

import (
	"flag"
	"fmt"
	"io"
	"os"
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
	if _, err := os.Stat(*repo); err != nil {
		return fmt.Errorf("run repository: %w", err)
	}
	if _, err := os.Stat(*config); err != nil {
		return fmt.Errorf("run config: %w", err)
	}
	_, err := fmt.Fprintf(output, "run accepted for %s\n", *repo)
	return err
}
