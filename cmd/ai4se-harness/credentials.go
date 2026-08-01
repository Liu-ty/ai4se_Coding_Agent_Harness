package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"golang.org/x/term"
)

func credentialsCommand(ctx context.Context, args []string, output, errors io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ai4se-harness credentials set|status|clear --provider <id> --endpoint <url>")
	}
	flags := flag.NewFlagSet("credentials "+args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	providerID := flags.String("provider", "", "provider ID")
	endpoint := flags.String("endpoint", "", "provider endpoint")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	ref, err := credentialRef(*providerID, *endpoint)
	if err != nil {
		return err
	}
	service := credentials.NewService(credentials.NewKeyringStore(), nil)
	switch args[0] {
	case "set":
		fmt.Fprint(errors, "API key: ")
		secret, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(errors)
		if err != nil {
			return fmt.Errorf("read API key: %w", err)
		}
		defer clear(secret)
		if err := service.Add(ctx, ref, secret); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "credential stored")
		return err
	case "status":
		status, err := service.Status(ctx, ref)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "configured: %t\n", status.Configured)
		return err
	case "clear":
		if err := service.Clear(ctx, ref); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "credential cleared")
		return err
	default:
		return fmt.Errorf("credentials command must be set, status, or clear")
	}
}

func credentialRef(providerID, endpoint string) (credentials.Ref, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return credentials.Ref{}, fmt.Errorf("endpoint must be an absolute URL")
	}
	return credentials.Ref{Provider: providerID, Host: parsed.Host}, nil
}

func clear(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
