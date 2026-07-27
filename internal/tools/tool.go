// Package tools contains bounded repository observation and mutation tools.
package tools

import (
	"context"
	"encoding/json"
)

type Result struct {
	Code      string          `json:"code"`
	Data      json.RawMessage `json:"data,omitempty"`
	Text      string          `json:"text,omitempty"`
	SHA256    string          `json:"sha256,omitempty"`
	Truncated bool            `json:"truncated"`
}

type Tool interface {
	Kind() string
	Execute(context.Context, json.RawMessage) (Result, error)
}
