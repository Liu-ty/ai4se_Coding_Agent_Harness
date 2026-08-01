package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var ErrApprovalWindowEnded = errors.New("approval window ended")

const (
	defaultApprovalTimeout = 15 * time.Minute
	ownedRunStopTimeout    = 5 * time.Second
	serverShutdownTimeout  = 2 * time.Second
)

func runCommand(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repo := flags.String("repo", "", "repository path")
	task := flags.String("task", "", "task text")
	config := flags.String("config", ".ai4se-harness.toml", "configuration file")
	approvalTimeout := flags.Duration("approval-timeout", defaultApprovalTimeout, "maximum local approval window")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *task == "" {
		return fmt.Errorf("run requires --repo and --task")
	}
	if *approvalTimeout <= 0 {
		return fmt.Errorf("run requires a positive --approval-timeout")
	}
	runtime, err := newLocalRuntime(ctx, *repo, localRuntimeOptions{})
	if err != nil {
		return fmt.Errorf("local composition: %w", err)
	}
	defer runtime.Close()
	run, err := runtime.Application.CreateRun(ctx, localRunRequest(*repo, *task, *config))
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "run created: %s\n", run.ID); err != nil {
		return err
	}
	if err := continueRun(ctx, runtime, run.ID, output, *approvalTimeout); err != nil {
		return err
	}
	return nil
}

// continueRun retains the creating runtime for the full run lifecycle. The
// in-memory loop controller, not SQLite, owns the session required to resume an
// approval. Once approval is required, the same application is exposed on a
// bounded ephemeral loopback server until the run is terminal.
func continueRun(
	ctx context.Context,
	runtime *localRuntime,
	runID domain.RunID,
	output io.Writer,
	approvalTimeout time.Duration,
) (returnErr error) {
	var server *ownedApprovalServer
	var approvalTimer *time.Timer
	var approvalDone <-chan time.Time
	defer func() {
		if approvalTimer != nil {
			approvalTimer.Stop()
		}
		if server != nil {
			returnErr = errors.Join(returnErr, server.Close())
		}
	}()

	for {
		run, err := runtime.Application.GetRun(ctx, runID)
		if err != nil {
			return stopOwnedRun(runtime, runID, err)
		}
		if terminalRunState(run.State) {
			return nil
		}
		if run.State == domain.StateAwaitingApproval && server == nil {
			server, err = startOwnedApprovalServer(runtime, runID, output, approvalTimeout)
			if err != nil {
				return stopOwnedRun(runtime, runID, fmt.Errorf("open approval server: %w", err))
			}
			approvalTimer = time.NewTimer(approvalTimeout)
			approvalDone = approvalTimer.C
		}
		if run.State != domain.StateAwaitingApproval && server != nil {
			if err := server.Close(); err != nil {
				return stopOwnedRun(runtime, runID, fmt.Errorf("close approval server: %w", err))
			}
			server = nil
		}
		if run.State != domain.StateAwaitingApproval && approvalTimer != nil {
			approvalTimer.Stop()
			approvalTimer = nil
			approvalDone = nil
		}

		select {
		case <-ctx.Done():
			return stopOwnedRun(runtime, runID, ctx.Err())
		case <-approvalDone:
			return stopOwnedRun(runtime, runID, fmt.Errorf(
				"%w after %s for %s", ErrApprovalWindowEnded, approvalTimeout, runID,
			))
		case err, open := <-serverErrors(server):
			if !open || err == nil {
				err = errors.New("approval server stopped without shutdown")
			}
			return stopOwnedRun(runtime, runID, fmt.Errorf("approval server stopped: %w", err))
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func terminalRunState(state domain.RunState) bool {
	return state == domain.StateSucceeded || state == domain.StateReviewComplete || state == domain.StateStopped
}

func stopOwnedRun(runtime *localRuntime, runID domain.RunID, cause error) error {
	stopCtx, cancel := context.WithTimeout(context.Background(), ownedRunStopTimeout)
	defer cancel()
	if err := runtime.Application.CancelRun(stopCtx, runID); err != nil {
		return errors.Join(cause, fmt.Errorf("stop owned run %s: %w", runID, err))
	}
	return fmt.Errorf("%w; run %s was stopped before runtime shutdown", cause, runID)
}

type ownedApprovalServer struct {
	server *http.Server
	errors chan error
}

func startOwnedApprovalServer(
	runtime *localRuntime,
	runID domain.RunID,
	output io.Writer,
	approvalTimeout time.Duration,
) (*ownedApprovalServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	address := listener.Addr().String()
	router, err := newLocalRouter(runtime, address)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	server := &ownedApprovalServer{
		server: &http.Server{Handler: router}, errors: make(chan error, 1),
	}
	go func() {
		serveErr := server.server.Serve(listener)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			server.errors <- serveErr
		}
		close(server.errors)
	}()
	if _, err := fmt.Fprintf(
		output,
		"run awaiting approval: %s\ncontinue at http://%s/?bootstrap=%s\napproval window: %s\n",
		runID, address, router.BootstrapToken(), approvalTimeout,
	); err != nil {
		_ = server.Close()
		return nil, err
	}
	return server, nil
}

func serverErrors(server *ownedApprovalServer) <-chan error {
	if server == nil {
		return nil
	}
	return server.errors
}

func (s *ownedApprovalServer) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	shutdownErr := s.server.Shutdown(shutdownCtx)
	if errors.Is(shutdownErr, context.DeadlineExceeded) {
		return s.server.Close()
	}
	return shutdownErr
}
