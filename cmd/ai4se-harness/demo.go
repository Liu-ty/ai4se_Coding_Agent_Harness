package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/demo"
)

func demoCommand(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || args[0] != "feedback-loop" {
		return fmt.Errorf("usage: ai4se-harness demo feedback-loop --format text|json")
	}
	flags := flag.NewFlagSet("demo feedback-loop", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	format := flags.String("format", "text", "output format: text or json")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	result, err := demo.RunFeedbackLoop(ctx)
	if err != nil {
		return err
	}
	switch *format {
	case "json":
		return json.NewEncoder(output).Encode(result)
	case "text":
		_, err := fmt.Fprintf(output, "state: %s\nevents: %v\n", result.State, result.Events)
		return err
	default:
		return fmt.Errorf("format must be text or json")
	}
}
