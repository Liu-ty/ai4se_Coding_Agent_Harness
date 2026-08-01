package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var ErrRunAwaitingApproval = errors.New("run is awaiting approval; continue it through the local server")

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
	if err := waitForTerminal(context.Background(), runtime.Application, run.ID); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "run created: %s\n", run.ID)
	return err
}

func waitForTerminal(ctx context.Context, application interface {
	GetRun(context.Context, domain.RunID) (domain.Run, error)
}, runID domain.RunID) error {
	for {
		run, err := application.GetRun(ctx, runID)
		if err != nil {
			return err
		}
		if run.State == domain.StateAwaitingApproval {
			return ErrRunAwaitingApproval
		}
		if run.State == domain.StateSucceeded || run.State == domain.StateReviewComplete || run.State == domain.StateStopped {
			return nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
