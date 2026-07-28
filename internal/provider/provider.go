package provider

import (
	"context"
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
type Response struct {
	Decision domain.AgentDecision
	Usage    Usage
}
type Provider interface {
	Decide(context.Context, Request) (Response, error)
}
