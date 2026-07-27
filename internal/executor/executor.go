package executor

import (
	"context"
	"errors"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var ErrUnsupportedPlatform = errors.New("unsupported executor platform: only Windows and Linux are supported")

const (
	CodeExit           = "EXIT"
	CodeStartError     = "START_ERROR"
	CodeExecutionError = "EXECUTION_ERROR"
	CodeTimeout        = "TIMEOUT"
	CodeCancelled      = "CANCELLED"
)

type Executor interface {
	Run(context.Context, config.CommandSpec) (domain.Observation, error)
}
