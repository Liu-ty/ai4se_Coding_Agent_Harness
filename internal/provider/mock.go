package provider

import (
	"context"
	"sync"
)

type MockCall struct {
	Request  Request
	Returned Response
}
type Mock struct {
	mu      sync.Mutex
	respond func(context.Context, Request) (Response, error)
	calls   []MockCall
}

func NewMock(respond func(context.Context, Request) (Response, error)) *Mock {
	return &Mock{respond: respond}
}
func (m *Mock) Decide(ctx context.Context, request Request) (Response, error) {
	response, err := m.respond(ctx, request)
	m.mu.Lock()
	m.calls = append(m.calls, MockCall{Request: request, Returned: response})
	m.mu.Unlock()
	return response, err
}
func (m *Mock) Calls() []MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MockCall, len(m.calls))
	copy(out, m.calls)
	return out
}
