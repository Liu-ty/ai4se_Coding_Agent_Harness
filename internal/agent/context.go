package agent

import "github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"

type ContextAssembler struct{ MaxItems int }

func (c ContextAssembler) Assemble(items []provider.ContextItem) []provider.ContextItem {
	max := c.MaxItems
	if max <= 0 {
		max = 16
	}
	if len(items) > max {
		items = items[len(items)-max:]
	}
	out := make([]provider.ContextItem, len(items))
	copy(out, items)
	return out
}
