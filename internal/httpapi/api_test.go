package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
)

const (
	localHost    = "127.0.0.1:4319"
	canarySecret = "TASK15_CANARY_SECRET_DO_NOT_RETURN_123456789"
)

type fakeApplication struct {
	mu        sync.Mutex
	runs      map[domain.RunID]domain.Run
	nextRun   domain.Run
	preflight app.PreflightReport
	createErr error
	cancelled domain.RunID
	approved  string
	rejected  string
	terminate bool
}

func (f *fakeApplication) CreateRun(_ context.Context, request app.CreateRunRequest) (domain.Run, error) {
	if f.createErr != nil {
		return domain.Run{}, f.createErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	run := f.nextRun
	if run.ID == "" {
		run = domain.Run{
			ID: "run-created", State: domain.StateCreated, Profile: request.Profile,
			Task: request.Task, RepoRoot: request.RepoRoot,
		}
	}
	if f.runs == nil {
		f.runs = make(map[domain.RunID]domain.Run)
	}
	f.runs[run.ID] = run
	return run, nil
}

func (f *fakeApplication) GetRun(_ context.Context, id domain.RunID) (domain.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	run, ok := f.runs[id]
	if !ok {
		return domain.Run{}, errors.New("missing run")
	}
	return run, nil
}

func (f *fakeApplication) CancelRun(_ context.Context, id domain.RunID) error {
	f.cancelled = id
	return nil
}

func (f *fakeApplication) Approve(_ context.Context, id domain.RunID, digest string) error {
	f.approved = string(id) + ":" + digest
	return nil
}

func (f *fakeApplication) Reject(_ context.Context, id domain.RunID, digest string, terminate bool) error {
	f.rejected = string(id) + ":" + digest
	f.terminate = terminate
	return nil
}

func (f *fakeApplication) Preflight(_ context.Context, _ app.CreateRunRequest) app.PreflightReport {
	return f.preflight
}

func TestLocalBootstrapIsOneTimeAndSetsStrictCookie(t *testing.T) {
	api, _, _ := newLocalAPI(t)
	token := api.BootstrapToken()
	first := httptest.NewRequest(http.MethodGet, "/?bootstrap="+token, nil)
	first.Host = localHost
	rr := httptest.NewRecorder()
	api.ServeHTTP(rr, first)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/" {
		t.Fatalf("bootstrap response = %d, location %q", rr.Code, rr.Header().Get("Location"))
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v", cookies)
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" {
		t.Fatalf("insecure cookie: %#v", cookie)
	}
	if cookie.Value == "" || strings.Contains(rr.Header().Get("Location"), token) {
		t.Fatal("session was not exchanged for a clean redirect")
	}

	second := httptest.NewRequest(http.MethodGet, "/?bootstrap="+token, nil)
	second.Host = localHost
	secondRR := httptest.NewRecorder()
	api.ServeHTTP(secondRR, second)
	assertAPIError(t, secondRR, http.StatusUnauthorized, "INVALID_BOOTSTRAP")
}

func TestLocalSecurityRejectsInvalidHostOriginSessionAndCSRF(t *testing.T) {
	api, _, _ := newLocalAPI(t)
	cookie := bootstrap(t, api)

	cases := []struct {
		name   string
		mutate func(*http.Request)
		code   string
	}{
		{name: "host", mutate: func(request *http.Request) { request.Host = "evil.example" }, code: "INVALID_HOST"},
		{name: "origin", mutate: func(request *http.Request) {
			request.Header.Set("Origin", "https://evil.example")
		}, code: "INVALID_ORIGIN"},
		{name: "session", mutate: func(request *http.Request) {
			request.Header.Del("Cookie")
		}, code: "SESSION_REQUIRED"},
		{name: "csrf", mutate: func(request *http.Request) {
			request.Header.Del(httpapi.CSRFHeader)
		}, code: "CSRF_REQUIRED"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := authorizedRequest(api, cookie, http.MethodPost, "/api/v1/runs", `{}`)
			test.mutate(request)
			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, request)
			assertAPIError(t, rr, http.StatusForbidden, test.code)
		})
	}
}

func TestLocalReadRejectsCrossOriginRequest(t *testing.T) {
	api, application, _ := newLocalAPI(t)
	application.runs = map[domain.RunID]domain.Run{"run-1": {ID: "run-1"}}
	cookie := bootstrap(t, api)
	request := authorizedRequest(api, cookie, http.MethodGet, "/api/v1/runs/run-1", "")
	request.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	api.ServeHTTP(rr, request)
	assertAPIError(t, rr, http.StatusForbidden, "INVALID_ORIGIN")
}

func TestRoutesUseVersionedJSONContracts(t *testing.T) {
	credentialAPI := credentials.NewService(credentials.NewMemoryStore(), nil)
	api, application, st := newLocalAPIWithCredentials(t, credentialAPI)
	cookie := bootstrap(t, api)
	application.runs = map[domain.RunID]domain.Run{
		"run-1": {ID: "run-1", State: domain.StateDeciding, Profile: domain.ProfileSupervised, Task: "repair"},
	}
	application.preflight = app.PreflightReport{
		OK: true, RepoRoot: "C:/repo", BaselineCommit: "abc", BaselineDiffHash: "def",
		Findings: []app.Finding{},
	}
	if err := st.CreateRun(context.Background(), domain.Run{ID: "run-1", State: domain.StateCreated}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutArtifact(context.Background(), domain.Artifact{
		ID: "diff-1", RunID: "run-1", Kind: "diff", SHA256: "sha",
		Content: []byte("patch " + canarySecret),
	}); err != nil {
		t.Fatal(err)
	}

	requests := []struct {
		method string
		path   string
		body   string
		status int
		check  func(*testing.T, []byte)
	}{
		{
			method: http.MethodPost, path: "/api/v1/runs",
			body:   `{"repo_root":"C:/repo","task":"repair","provider":"openai","model":"m","endpoint":"https://api.openai.com","profile":"supervised"}`,
			status: http.StatusCreated,
			check:  func(t *testing.T, body []byte) { assertJSONContains(t, body, `"id":"run-created"`) },
		},
		{
			method: http.MethodGet, path: "/api/v1/runs/run-1", status: http.StatusOK,
			check: func(t *testing.T, body []byte) { assertJSONContains(t, body, `"state":"DECIDING"`) },
		},
		{method: http.MethodPost, path: "/api/v1/runs/run-1/cancel", body: `{}`, status: http.StatusNoContent},
		{method: http.MethodPost, path: "/api/v1/runs/run-1/approvals/digest-a/approve", body: `{}`, status: http.StatusNoContent},
		{
			method: http.MethodPost, path: "/api/v1/runs/run-1/approvals/digest-b/reject",
			body: `{"terminate":true}`, status: http.StatusNoContent,
		},
		{
			method: http.MethodGet, path: "/api/v1/runs/run-1/artifacts/diff-1", status: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				assertJSONContains(t, body, `"id":"diff-1"`)
				var response struct {
					Content []byte `json:"content"`
				}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(response.Content, []byte(canarySecret)) ||
					!bytes.Contains(response.Content, []byte("[REDACTED]")) {
					t.Fatalf("artifact content leaked: %q", response.Content)
				}
			},
		},
		{
			method: http.MethodPost, path: "/api/v1/config/validate",
			body:   `{"repo_root":"C:/repo","provider":"openai","endpoint":"https://api.openai.com","profile":"review"}`,
			status: http.StatusOK,
			check:  func(t *testing.T, body []byte) { assertJSONContains(t, body, `"baseline_commit":"abc"`) },
		},
		{
			method: http.MethodPut, path: "/api/v1/credentials/openai/api.openai.com",
			body: `{"secret":"credential-value"}`, status: http.StatusNoContent,
		},
		{
			method: http.MethodGet, path: "/api/v1/credentials/openai/api.openai.com", status: http.StatusOK,
			check: func(t *testing.T, body []byte) {
				assertJSONContains(t, body, `"configured":true`)
				if bytes.Contains(body, []byte("credential-value")) {
					t.Fatalf("credential leaked: %s", body)
				}
			},
		},
		{
			method: http.MethodDelete, path: "/api/v1/credentials/openai/api.openai.com",
			status: http.StatusNoContent,
		},
	}

	for _, test := range requests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := authorizedRequest(api, cookie, test.method, test.path, test.body)
			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, request)
			if rr.Code != test.status {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			if test.check != nil {
				test.check(t, rr.Body.Bytes())
			}
		})
	}

	if application.cancelled != "run-1" ||
		application.approved != "run-1:digest-a" ||
		application.rejected != "run-1:digest-b" || !application.terminate {
		t.Fatalf("mutation calls were not routed: %#v", application)
	}
}

func TestLocalConstructorRequiresIPv4LoopbackBinding(t *testing.T) {
	for _, host := range []string{"localhost:4319", "0.0.0.0:4319", "[::1]:4319"} {
		t.Run(host, func(t *testing.T) {
			_, err := httpapi.NewLocal(httpapi.Options{
				Application:  &fakeApplication{},
				Store:        store.NewMemory(),
				Capabilities: httpapi.LocalCapabilities(),
				Credentials:  credentials.NewService(credentials.NewMemoryStore(), nil),
				Host:         host,
			})
			if err == nil {
				t.Fatalf("accepted non-127.0.0.1 bind host %q", host)
			}
		})
	}
}

func TestLocalConstructorRejectsCollidingSecurityTokens(t *testing.T) {
	_, err := httpapi.NewLocal(httpapi.Options{
		Application:  &fakeApplication{},
		Store:        store.NewMemory(),
		Capabilities: httpapi.LocalCapabilities(),
		Credentials:  credentials.NewService(credentials.NewMemoryStore(), nil),
		Host:         localHost,
		Random:       bytes.NewReader(bytes.Repeat([]byte{0x42}, 96)),
	})
	if err == nil {
		t.Fatal("accepted colliding bootstrap, session, and CSRF tokens")
	}
}

func TestInvalidJSONBodyLimitAndErrorsAreUniformBoundedAndRedacted(t *testing.T) {
	api, application, _ := newLocalAPI(t)
	cookie := bootstrap(t, api)

	invalid := authorizedRequest(api, cookie, http.MethodPost, "/api/v1/runs", `{`)
	invalidRR := httptest.NewRecorder()
	api.ServeHTTP(invalidRR, invalid)
	assertAPIError(t, invalidRR, http.StatusBadRequest, "INVALID_JSON")
	assertJSONContains(t, invalidRR.Body.Bytes(), `"message":"request body is not valid JSON"`)

	oversized := authorizedRequest(
		api, cookie, http.MethodPost, "/api/v1/runs",
		`{"task":"`+strings.Repeat("x", 2048)+`"}`,
	)
	oversizedRR := httptest.NewRecorder()
	api.ServeHTTP(oversizedRR, oversized)
	assertAPIError(t, oversizedRR, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE")

	application.createErr = errors.New(canarySecret + strings.Repeat("z", 4096))
	failed := authorizedRequest(api, cookie, http.MethodPost, "/api/v1/runs", `{}`)
	failedRR := httptest.NewRecorder()
	api.ServeHTTP(failedRR, failed)
	assertAPIError(t, failedRR, http.StatusInternalServerError, "INTERNAL_ERROR")
	if failedRR.Body.Len() > 1024 || strings.Contains(failedRR.Body.String(), canarySecret) {
		t.Fatalf("unbounded or secret-bearing error: %q", failedRR.Body.String())
	}

	unknown := authorizedRequest(api, cookie, http.MethodGet, "/api/v1/unknown", "")
	unknownRR := httptest.NewRecorder()
	api.ServeHTTP(unknownRR, unknown)
	assertAPIError(t, unknownRR, http.StatusNotFound, "NOT_FOUND")
}

func TestSSEReplaysSequenceAndTypeAfterLastEventIDAndRedactsCanary(t *testing.T) {
	api, _, st := newLocalAPI(t)
	cookie := bootstrap(t, api)
	ctx := context.Background()
	if err := st.CreateRun(ctx, domain.Run{ID: "run-events", State: domain.StateCreated}); err != nil {
		t.Fatal(err)
	}
	for _, seeded := range []struct {
		eventType string
		payload   string
	}{
		{"One", `{"value":"one"}`},
		{"Two", `{"value":"two"}`},
		{"Three-" + canarySecret, `{"value":"` + canarySecret + `"}`},
	} {
		if _, err := st.AppendEvent(ctx, "run-events", seeded.eventType, json.RawMessage(seeded.payload)); err != nil {
			t.Fatal(err)
		}
	}

	requestCtx, cancel := context.WithCancel(context.Background())
	request := authorizedRequest(api, cookie, http.MethodGet, "/api/v1/runs/run-events/events", "")
	request = request.WithContext(requestCtx)
	request.Header.Set("Last-Event-ID", "1")
	writer := newCancellingWriter(cancel, "id: 3")
	done := make(chan struct{})
	go func() {
		api.ServeHTTP(writer, request)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not stop after disconnect")
	}
	body := writer.String()
	if strings.Contains(body, "id: 1") || !strings.Contains(body, "id: 2") ||
		!strings.Contains(body, "event: Two") || !strings.Contains(body, "id: 3") ||
		!strings.Contains(body, "event: Three") {
		t.Fatalf("unexpected replay:\n%s", body)
	}
	if strings.Contains(body, canarySecret) || !strings.Contains(body, "[REDACTED]") {
		t.Fatalf("event was not redacted:\n%s", body)
	}
}

func TestSSEHeartbeatAndMalformedLastEventID(t *testing.T) {
	api, _, st := newLocalAPI(t)
	cookie := bootstrap(t, api)
	if err := st.CreateRun(context.Background(), domain.Run{ID: "run-heartbeat"}); err != nil {
		t.Fatal(err)
	}

	badContext, badCancel := context.WithCancel(context.Background())
	badCancel()
	bad := authorizedRequest(api, cookie, http.MethodGet, "/api/v1/runs/run-heartbeat/events", "")
	bad = bad.WithContext(badContext)
	bad.Header.Set("Last-Event-ID", "not-a-sequence")
	badRR := httptest.NewRecorder()
	api.ServeHTTP(badRR, bad)
	assertAPIError(t, badRR, http.StatusBadRequest, "INVALID_LAST_EVENT_ID")

	requestCtx, cancel := context.WithCancel(context.Background())
	request := authorizedRequest(api, cookie, http.MethodGet, "/api/v1/runs/run-heartbeat/events", "")
	request = request.WithContext(requestCtx)
	writer := newCancellingWriter(cancel, ": heartbeat")
	done := make(chan struct{})
	go func() {
		api.ServeHTTP(writer, request)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat was not emitted or disconnect was ignored")
	}
	if !strings.Contains(writer.String(), ": heartbeat\n\n") {
		t.Fatalf("heartbeat missing: %q", writer.String())
	}
}

func TestCredentialMutationCanaryIsRedactedFromLaterEvents(t *testing.T) {
	const submittedSecret = "ordinary-credential-value-987654321"
	api, _, st := newLocalAPI(t)
	cookie := bootstrap(t, api)
	put := authorizedRequest(
		api, cookie, http.MethodPut,
		"/api/v1/credentials/openai/api.openai.com",
		`{"secret":"`+submittedSecret+`"}`,
	)
	putRR := httptest.NewRecorder()
	api.ServeHTTP(putRR, put)
	if putRR.Code != http.StatusNoContent {
		t.Fatalf("credential PUT = %d, body = %s", putRR.Code, putRR.Body.String())
	}
	if err := st.CreateRun(context.Background(), domain.Run{ID: "run-dynamic-secret"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent(
		context.Background(), "run-dynamic-secret", "Observation",
		json.RawMessage(`{"value":"`+submittedSecret+`"}`),
	); err != nil {
		t.Fatal(err)
	}

	requestCtx, cancel := context.WithCancel(context.Background())
	request := authorizedRequest(
		api, cookie, http.MethodGet,
		"/api/v1/runs/run-dynamic-secret/events", "",
	).WithContext(requestCtx)
	writer := newCancellingWriter(cancel, "id: 1")
	api.ServeHTTP(writer, request)
	body := writer.String()
	if strings.Contains(body, submittedSecret) || !strings.Contains(body, "[REDACTED]") {
		t.Fatalf("submitted credential leaked through event: %s", body)
	}
}

func TestDemoCapabilitiesPruneLocalAndNonFixedRoutes(t *testing.T) {
	application := &fakeApplication{runs: map[domain.RunID]domain.Run{
		"fixed": {ID: "fixed", State: domain.StateSucceeded},
	}}
	st := store.NewMemory()
	if err := st.CreateRun(context.Background(), domain.Run{ID: "fixed"}); err != nil {
		t.Fatal(err)
	}
	overbroad := httpapi.LocalCapabilities()
	overbroad.FixedRuns = map[domain.RunID]struct{}{"fixed": {}}
	api, err := httpapi.NewDemo(httpapi.Options{
		Application:       application,
		Store:             st,
		Credentials:       credentials.NewService(credentials.NewMemoryStore(), nil),
		Capabilities:      overbroad,
		HeartbeatInterval: 5 * time.Millisecond,
		PollInterval:      2 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/runs"},
		{http.MethodPost, "/api/v1/config/validate"},
		{http.MethodGet, "/api/v1/credentials/openai/api.openai.com"},
		{http.MethodGet, "/api/v1/runs/fixed/artifacts/a"},
		{http.MethodGet, "/api/v1/runs/not-fixed"},
		{http.MethodGet, "/api/v1/runs/not-fixed/events"},
	} {
		rr := httptest.NewRecorder()
		api.ServeHTTP(rr, httptest.NewRequest(test.method, test.path, nil))
		assertAPIError(t, rr, http.StatusNotFound, "NOT_FOUND")
	}

	rr := httptest.NewRecorder()
	api.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs/fixed", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("fixed demo run status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestHealthzIsPublicAndCORSIsNeverEnabled(t *testing.T) {
	api, _, _ := newLocalAPI(t)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Host = localHost
	request.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	api.ServeHTTP(rr, request)
	if rr.Code != http.StatusOK || rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("health response = %d, headers = %#v", rr.Code, rr.Header())
	}
	assertJSONContains(t, rr.Body.Bytes(), `"status":"ok"`)
}

func newLocalAPI(t *testing.T) (*httpapi.Router, *fakeApplication, *store.MemoryStore) {
	t.Helper()
	return newLocalAPIWithCredentials(t, credentials.NewService(credentials.NewMemoryStore(), nil))
}

func newLocalAPIWithCredentials(
	t *testing.T,
	credentialAPI *credentials.Service,
) (*httpapi.Router, *fakeApplication, *store.MemoryStore) {
	t.Helper()
	application := &fakeApplication{}
	st := store.NewMemory()
	api, err := httpapi.NewLocal(httpapi.Options{
		Application:       application,
		Store:             st,
		Credentials:       credentialAPI,
		Capabilities:      httpapi.LocalCapabilities(),
		Host:              localHost,
		MaxBodyBytes:      1024,
		HeartbeatInterval: 5 * time.Millisecond,
		PollInterval:      2 * time.Millisecond,
		Secrets:           []string{canarySecret},
		Random:            bytes.NewReader(deterministicRandomBytes(4096)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return api, application, st
}

func deterministicRandomBytes(size int) []byte {
	value := make([]byte, size)
	for index := range value {
		value[index] = byte(index / 32)
	}
	return value
}

func bootstrap(t *testing.T, api *httpapi.Router) *http.Cookie {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/?bootstrap="+api.BootstrapToken(), nil)
	request.Host = localHost
	rr := httptest.NewRecorder()
	api.ServeHTTP(rr, request)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("bootstrap status = %d, body = %s", rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("bootstrap cookies = %#v", cookies)
	}
	return cookies[0]
}

func authorizedRequest(
	api *httpapi.Router,
	cookie *http.Cookie,
	method string,
	path string,
	body string,
) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Host = localHost
	request.Header.Set("Origin", "http://"+localHost)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(httpapi.CSRFHeader, api.CSRFToken())
	request.AddCookie(cookie)
	return request
}

func assertAPIError(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var envelope struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid error JSON: %v (%q)", err, rr.Body.String())
	}
	if envelope.Error.Code != code || envelope.Error.Message == "" || envelope.Error.RequestID == "" {
		t.Fatalf("error envelope = %#v", envelope)
	}
}

func assertJSONContains(t *testing.T, body []byte, value string) {
	t.Helper()
	if !bytes.Contains(body, []byte(value)) {
		t.Fatalf("%q missing from %s", value, body)
	}
}

type cancellingWriter struct {
	mu      sync.Mutex
	header  http.Header
	body    strings.Builder
	cancel  context.CancelFunc
	trigger string
}

func newCancellingWriter(cancel context.CancelFunc, trigger string) *cancellingWriter {
	return &cancellingWriter{
		header: make(http.Header), cancel: cancel, trigger: trigger,
	}
}

func (w *cancellingWriter) Header() http.Header {
	return w.header
}

func (w *cancellingWriter) WriteHeader(_ int) {}

func (w *cancellingWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.body.Write(data)
	if strings.Contains(w.body.String(), w.trigger) {
		w.cancel()
	}
	return len(data), nil
}

func (w *cancellingWriter) Flush() {}

func (w *cancellingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}
