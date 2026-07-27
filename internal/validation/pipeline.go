package validation

import (
	"context"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/executor"
)

type Pipeline struct {
	stages   []config.CommandSpec
	executor executor.Executor
}

func New(stages []config.CommandSpec, ex executor.Executor) *Pipeline {
	copied := make([]config.CommandSpec, len(stages))
	copy(copied, stages)
	return &Pipeline{stages: copied, executor: ex}
}

func (p *Pipeline) RunStage(ctx context.Context, index int) StageResult {
	if index < 0 || index >= len(p.stages) {
		return StageResult{}
	}
	stage := p.stages[index]
	obs, err := p.executor.Run(ctx, stage)
	if err != nil && obs.Code == "" {
		obs.Code = executor.CodeStartError
	}
	return StageResult{
		Stage:       stage,
		Observation: obs,
		Passed:      passed(obs),
	}
}

func (p *Pipeline) RunFrom(ctx context.Context, start int) Result {
	return p.run(ctx, start, false)
}

func (p *Pipeline) RunAllRequired(ctx context.Context) Result {
	return p.run(ctx, 0, true)
}

func (p *Pipeline) run(ctx context.Context, start int, requiredOnly bool) Result {
	var result Result
	if start < 0 {
		start = 0
	}
	for i := start; i < len(p.stages); i++ {
		if requiredOnly && !p.stages[i].Required {
			continue
		}
		stageResult := p.RunStage(ctx, i)
		result.Stages = append(result.Stages, stageResult)
		if !stageResult.Passed && stageResult.Stage.Required {
			result.FailedStage = stageResult.Stage.ID
			return result
		}
	}
	result.Complete = true
	return result
}

func passed(obs domain.Observation) bool {
	return obs.Code == executor.CodeExit && obs.ExitCode != nil && *obs.ExitCode == 0
}
