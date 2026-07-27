package validation

import (
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

type StageResult struct {
	Stage       config.CommandSpec
	Observation domain.Observation
	Passed      bool
}

type Result struct {
	Stages      []StageResult
	FailedStage string
	Complete    bool
}
