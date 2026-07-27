package executor

import (
	"context"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

const (
	CodeExit       = "EXIT"
	CodeStartError = "START_ERROR"
	CodeTimeout    = "TIMEOUT"
	CodeCancelled  = "CANCELLED"
)

type Executor interface {
	Run(context.Context, config.CommandSpec) (domain.Observation, error)
}
