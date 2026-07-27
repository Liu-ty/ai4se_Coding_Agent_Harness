package validation_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/validation"
)

func TestRunFromStopsAtFirstRequiredFailure(t *testing.T) {
	ex := &executor.Mock{Results: map[string][]domain.Observation{
		"targeted": {exit(0)},
		"full":     {exit(1)},
		"lint":     {exit(0)},
	}}
	p := validation.New(stages("targeted", "full", "lint"), ex)

	got := p.RunFrom(context.Background(), 0)

	if got.FailedStage != "full" || got.Complete {
		t.Fatalf("result = %#v", got)
	}
	if callIDs(ex.Calls) != "targeted,full" {
		t.Fatalf("calls = %s", callIDs(ex.Calls))
	}
}

func TestRunAllRequiredRerunsEveryRequiredStage(t *testing.T) {
	ex := &executor.Mock{Results: map[string][]domain.Observation{
		"targeted": {exit(0)},
		"full":     {exit(0)},
		"lint":     {exit(0)},
	}}
	p := validation.New(stages("targeted", "full", "lint"), ex)

	got := p.RunAllRequired(context.Background())

	if !got.Complete || got.FailedStage != "" {
		t.Fatalf("result = %#v", got)
	}
	if callIDs(ex.Calls) != "targeted,full,lint" {
		t.Fatalf("calls = %s", callIDs(ex.Calls))
	}
}

func TestOptionalFailureIsRecordedWithoutStoppingRequiredStages(t *testing.T) {
	stageList := stages("targeted", "advisory", "full")
	stageList[1].Required = false
	ex := &executor.Mock{Results: map[string][]domain.Observation{
		"targeted": {exit(0)},
		"advisory": {exit(1)},
		"full":     {exit(0)},
	}}
	p := validation.New(stageList, ex)

	got := p.RunFrom(context.Background(), 0)

	if !got.Complete || got.FailedStage != "" {
		t.Fatalf("result = %#v", got)
	}
	if len(got.Stages) != 3 || got.Stages[1].Passed {
		t.Fatalf("stage results = %#v", got.Stages)
	}
	if callIDs(ex.Calls) != "targeted,advisory,full" {
		t.Fatalf("calls = %s", callIDs(ex.Calls))
	}
}

func TestRunStageTreatsMissingExitCodeAsFailure(t *testing.T) {
	ex := &executor.Mock{Results: map[string][]domain.Observation{
		"targeted": {{Code: executor.CodeExit}},
	}}
	p := validation.New(stages("targeted"), ex)

	got := p.RunStage(context.Background(), 0)

	if got.Passed {
		t.Fatalf("stage passed with missing exit code: %#v", got)
	}
}

func TestContextCancellationStopsPipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ex := &executor.Mock{}
	p := validation.New(stages("targeted", "full"), ex)

	got := p.RunFrom(ctx, 0)

	if got.Complete || got.FailedStage != "targeted" || callIDs(ex.Calls) != "targeted" {
		t.Fatalf("result=%#v calls=%s", got, callIDs(ex.Calls))
	}
}

func stages(ids ...string) []config.CommandSpec {
	out := make([]config.CommandSpec, 0, len(ids))
	for _, id := range ids {
		out = append(out, config.CommandSpec{ID: id, Required: true})
	}
	return out
}

func exit(code int) domain.Observation {
	return domain.Observation{Code: executor.CodeExit, ExitCode: &code}
}

func callIDs(calls []config.CommandSpec) string {
	ids := make([]string, 0, len(calls))
	for _, call := range calls {
		ids = append(ids, call.ID)
	}
	return strings.Join(ids, ",")
}
