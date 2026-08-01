package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
)

func serve(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profile := flags.String("profile", "", "local or demo")
	repo := flags.String("repo", "", "repository path")
	address := flags.String("addr", "", "listen address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	switch *profile {
	case "demo":
		if *address == "" {
			*address = "0.0.0.0:8080"
		}
		composition, err := demo.NewComposition(ctx, *address)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "demo listening on http://%s\n", *address)
		return serveHTTP(ctx, &http.Server{
			Addr: *address, Handler: composition.Router(), ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
		})
	case "local":
		if *address == "" {
			*address = "127.0.0.1:4319"
		}
		if *repo == "" {
			return fmt.Errorf("serve --profile local requires --repo")
		}
		if info, err := os.Stat(*repo); err != nil || !info.IsDir() {
			return fmt.Errorf("local repository is not a directory")
		}
		return serveLocal(ctx, *repo, *address, output)
	default:
		return fmt.Errorf("profile must be local or demo")
	}
}

func serveLocal(ctx context.Context, repo, address string, output io.Writer) error {
	runtime, err := newLocalRuntime(ctx, repo, localRuntimeOptions{})
	if err != nil {
		return err
	}
	defer runtime.Close()
	router, err := newLocalRouter(runtime, address)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "local listening on http://%s/?bootstrap=%s\n", address, router.BootstrapToken())
	return serveHTTP(ctx, &http.Server{
		Addr: address, Handler: router, ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	})
}

func serveHTTP(ctx context.Context, server *http.Server) error {
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- server.ListenAndServe() }()
	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()
		err := server.Shutdown(shutdownCtx)
		if errors.Is(err, context.DeadlineExceeded) {
			err = server.Close()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		if serveErr := <-errorsCh; serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return nil
	}
}

func newLocalRouter(runtime *localRuntime, address string) (*httpapi.Router, error) {
	return httpapi.NewLocal(httpapi.Options{
		Application: runtime.Application, Store: runtime.Store, Credentials: runtime.Credentials,
		Capabilities: httpapi.LocalCapabilities(), AppShell: httpapi.WebHandler(), Host: address,
	})
}
