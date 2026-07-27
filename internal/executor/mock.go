package executor

import (
	"context"
	"sync"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

type Mock struct {
	mu      sync.Mutex
	Results map[string][]domain.Observation
	Calls   []config.CommandSpec
}

func (m *Mock) Run(ctx context.Context, spec config.CommandSpec) (domain.Observation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, spec)
	if err := ctx.Err(); err != nil {
		return domain.Observation{Code: CodeCancelled}, nil
	}
	queue := m.Results[spec.ID]
	if len(queue) == 0 {
		zero := 0
		return domain.Observation{Code: CodeExit, ExitCode: &zero}, nil
	}
	got := queue[0]
	m.Results[spec.ID] = queue[1:]
	return got, nil
}
