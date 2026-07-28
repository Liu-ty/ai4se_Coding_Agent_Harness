package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type Anthropic struct {
	*httpProvider
	options Options
}

func NewAnthropic(endpoint string, client *http.Client, credentials CredentialSource, options Options) (*Anthropic, error) {
	h, err := newHTTPProvider(endpoint, client, credentials, "anthropic")
	if err != nil {
		return nil, err
	}
	return &Anthropic{httpProvider: h, options: options}, nil
}
func (p *Anthropic) Decide(ctx context.Context, request Request) (Response, error) {
	payload := map[string]any{"model": p.options.Model, "max_tokens": p.options.MaxTokens, "system": "Return one canonical JSON AgentDecision.", "messages": []map[string]any{{"role": "user", "content": request}}}
	b, _ := json.Marshal(payload)
	data, err := p.post(ctx, "/v1/messages", bytes.NewReader(b), func(h http.Header, key []byte) {
		h.Set("x-api-key", string(key))
		h.Set("anthropic-version", "2023-06-01")
	})
	if err != nil {
		return Response{}, err
	}
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			Input  int `json:"input_tokens"`
			Output int `json:"output_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(data, &out) != nil || len(out.Content) == 0 {
		return Response{}, ErrInvalidResponse
	}
	var d Response
	if out.Content[0].Type != "text" || json.Unmarshal([]byte(out.Content[0].Text), &d.Decision) != nil || !validDecision(d.Decision) {
		return Response{}, ErrInvalidResponse
	}
	d.Usage = Usage{InputTokens: out.Usage.Input, OutputTokens: out.Usage.Output}
	return d, nil
}
