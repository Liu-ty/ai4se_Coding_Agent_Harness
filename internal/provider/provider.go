package provider

import (
	"context"
	"encoding/json"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

type ContextItem struct{ Kind, Label, Content, SHA256 string }
type Request struct {
	Task           string
	Context        []ContextItem
	LastFeedback   *domain.StructuredFeedback
	AllowedActions []string
}
type Usage struct{ InputTokens, OutputTokens int }
type Options struct {
	Model     string
	MaxTokens int
}
type Response struct {
	Decision domain.AgentDecision
	Usage    Usage
}
type Provider interface {
	Decide(context.Context, Request) (Response, error)
}

func validDecision(d domain.AgentDecision) bool {
	return d.Version == "1" && d.Action.Kind != "" && json.Valid(d.Action.Args)
}
