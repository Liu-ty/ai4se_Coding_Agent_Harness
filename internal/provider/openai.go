package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type OpenAI struct {
	*httpProvider
	options Options
}

func NewOpenAI(endpoint string, client *http.Client, credentials CredentialSource, options Options) (*OpenAI, error) {
	h, err := newHTTPProvider(endpoint, client, credentials, "openai", options)
	if err != nil {
		return nil, err
	}
	return &OpenAI{httpProvider: h, options: options}, nil
}
func (p *OpenAI) Decide(ctx context.Context, request Request) (Response, error) {
	payload := map[string]any{"model": p.options.Model, "messages": []map[string]any{{"role": "system", "content": "Return one canonical JSON AgentDecision."}, {"role": "user", "content": request}}}
	b, _ := json.Marshal(payload)
	data, err := p.post(ctx, "/v1/chat/completions", bytes.NewReader(b), func(h http.Header, key []byte) { h.Set("Authorization", "Bearer "+string(key)) })
	if err != nil {
		return Response{}, err
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Input  int `json:"prompt_tokens"`
			Output int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(data, &out) != nil || len(out.Choices) == 0 {
		return Response{}, ErrInvalidResponse
	}
	var d Response
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &d.Decision); err != nil || !validDecision(d.Decision) {
		return Response{}, ErrInvalidResponse
	}
	d.Usage = Usage{InputTokens: out.Usage.Input, OutputTokens: out.Usage.Output}
	return d, nil
}
