package provider

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBytes int64 = 1 << 20
const defaultTimeout = 30 * time.Second

var ErrAuthentication = errors.New("provider authentication failed")
var ErrRateLimited = errors.New("provider rate limited")
var ErrInvalidResponse = errors.New("provider invalid response")
var ErrTransport = errors.New("provider transport failed")

type CredentialSource interface {
	// Get returns an independently owned buffer that the provider clears after
	// copying it into request-scoped memory.
	Get(context.Context, string, string) ([]byte, error)
}
type httpProvider struct {
	endpoint    *url.URL
	client      *http.Client
	credentials CredentialSource
	kind        string
}

func newHTTPProvider(endpoint string, client *http.Client, credentials CredentialSource, kind string, options Options) (*httpProvider, error) {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !safeCredentialTransport(u) {
		return nil, ErrTransport
	}
	if !knownEndpoint(kind, u) && !options.ConfirmCustomEndpoint {
		return nil, ErrTransport
	}
	if client == nil {
		client = http.DefaultClient
	}
	clone := *client
	if clone.Timeout <= 0 {
		clone.Timeout = defaultTimeout
	}
	origin := u.Host
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != origin || req.URL.Scheme != u.Scheme {
			return http.ErrUseLastResponse
		}
		return nil
	}
	return &httpProvider{endpoint: u, client: &clone, credentials: credentials, kind: kind}, nil
}

func safeCredentialTransport(endpoint *url.URL) bool {
	if endpoint.Scheme == "https" {
		return true
	}
	if endpoint.Scheme != "http" {
		return false
	}
	ip := net.ParseIP(endpoint.Hostname())
	return ip != nil && ip.IsLoopback()
}

func knownEndpoint(kind string, endpoint *url.URL) bool {
	if endpoint.Port() != "" && endpoint.Port() != "443" {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(endpoint.Hostname()), ".")
	switch kind {
	case "openai":
		return host == "api.openai.com"
	case "anthropic":
		return host == "api.anthropic.com"
	default:
		return false
	}
}
func (p *httpProvider) post(ctx context.Context, path string, body io.Reader, headers func(http.Header, []byte)) ([]byte, error) {
	if p.endpoint == nil || p.endpoint.Scheme == "" || p.endpoint.Host == "" {
		return nil, ErrTransport
	}
	sourceKey, err := p.credentials.Get(ctx, p.kind, p.endpoint.Host)
	if err != nil {
		return nil, ErrAuthentication
	}
	key := append([]byte(nil), sourceKey...)
	for i := range sourceKey {
		sourceKey[i] = 0
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	u := *p.endpoint
	basePath := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(path, "/v1/") {
		u.Path = basePath + strings.TrimPrefix(path, "/v1")
	} else if basePath != "" && strings.HasPrefix(path, basePath+"/") {
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
