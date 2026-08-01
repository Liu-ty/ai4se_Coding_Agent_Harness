package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
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
		return http.ListenAndServe(*address, composition.Router())
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
		return serveLocal(*address, output)
	default:
		return fmt.Errorf("profile must be local or demo")
	}
}

func serveLocal(address string, output io.Writer) error {
	application := localApplication{store: store.NewMemory()}
	router, err := httpapi.NewLocal(httpapi.Options{
		Application: &application, Store: application.store,
		Credentials:  credentials.NewService(credentials.NewKeyringStore(), nil),
		Capabilities: httpapi.LocalCapabilities(), AppShell: httpapi.WebHandler(), Host: address,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "local listening on http://%s/?bootstrap=%s\n", address, router.BootstrapToken())
	return http.ListenAndServe(address, router)
}

type localApplication struct{ store *store.MemoryStore }

func (*localApplication) CreateRun(context.Context, app.CreateRunRequest) (domain.Run, error) {
	return domain.Run{}, errors.New("local run composition requires configured provider")
}
func (a *localApplication) GetRun(ctx context.Context, id domain.RunID) (domain.Run, error) {
	return a.store.GetRun(ctx, id)
}
func (*localApplication) CancelRun(context.Context, domain.RunID) error {
	return errors.New("local run composition requires configured provider")
}
func (*localApplication) Approve(context.Context, domain.RunID, string) error {
	return errors.New("local run composition requires configured provider")
}
func (*localApplication) Reject(context.Context, domain.RunID, string, bool) error {
	return errors.New("local run composition requires configured provider")
}
func (*localApplication) Preflight(context.Context, app.CreateRunRequest) app.PreflightReport {
	return app.PreflightReport{}
}
