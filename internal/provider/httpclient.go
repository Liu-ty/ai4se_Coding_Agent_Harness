package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxResponseBytes int64 = 1 << 20

var ErrAuthentication = errors.New("provider authentication failed")
var ErrRateLimited = errors.New("provider rate limited")
var ErrInvalidResponse = errors.New("provider invalid response")
var ErrTransport = errors.New("provider transport failed")

type CredentialSource interface {
	Get(context.Context, string, string) ([]byte, error)
}
type httpProvider struct {
	endpoint    *url.URL
	client      *http.Client
	credentials CredentialSource
	kind        string
}

func newHTTPProvider(endpoint string, client *http.Client, credentials CredentialSource, kind string) *httpProvider {
	u, _ := url.Parse(endpoint)
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	origin := u.Host
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != origin {
			return http.ErrUseLastResponse
		}
		return nil
	}
	return &httpProvider{endpoint: u, client: &clone, credentials: credentials, kind: kind}
}
func (p *httpProvider) post(ctx context.Context, path string, body io.Reader, headers func(http.Header, []byte)) ([]byte, error) {
	if p.endpoint == nil || p.endpoint.Scheme == "" || p.endpoint.Host == "" {
		return nil, ErrTransport
	}
	key, err := p.credentials.Get(ctx, p.kind, p.endpoint.Host)
	if err != nil {
		return nil, ErrAuthentication
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	u := *p.endpoint
	basePath := strings.TrimRight(u.Path, "/")
	if basePath != "" && strings.HasPrefix(path, basePath+"/") {
		u.Path = path
	} else {
		u.Path = basePath + path
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return nil, ErrTransport
	}
	req.Header.Set("Content-Type", "application/json")
	headers(req.Header, key)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ErrTransport
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || int64(len(data)) > maxResponseBytes {
		return nil, ErrInvalidResponse
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrAuthentication
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ErrTransport
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return nil, ErrInvalidResponse
	}
	return data, nil
}
