package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

const (
	CSRFHeader        = "X-CSRF-Token"
	sessionCookieName = "ai4se_session"
	defaultBodyLimit  = int64(1 << 20)
	maxErrorMessage   = 512
)

type Application interface {
	CreateRun(context.Context, app.CreateRunRequest) (domain.Run, error)
	GetRun(context.Context, domain.RunID) (domain.Run, error)
	CancelRun(context.Context, domain.RunID) error
	Approve(context.Context, domain.RunID, string) error
	Reject(context.Context, domain.RunID, string, bool) error
	Preflight(context.Context, app.CreateRunRequest) app.PreflightReport
}

type Store interface {
	ListRuns(context.Context, storeport.RunListQuery) (storeport.RunPage, error)
	ListEvents(context.Context, domain.RunID, uint64) ([]domain.RunEvent, error)
	GetArtifact(context.Context, domain.RunID, string) (domain.Artifact, error)
}

type CredentialService interface {
	Add(context.Context, credentials.Ref, []byte) error
	Update(context.Context, credentials.Ref, []byte) error
	Status(context.Context, credentials.Ref) (credentials.Status, error)
	Clear(context.Context, credentials.Ref) error
}

type Capabilities struct {
	CreateRuns       bool
	ListRuns         bool
	ReadRuns         bool
	CancelRuns       bool
	Approvals        bool
	Events           bool
	Artifacts        bool
	ConfigValidation bool
	Credentials      bool
	FixedRuns        map[domain.RunID]struct{}
}

func LocalCapabilities() Capabilities {
	return Capabilities{
		CreateRuns: true, ListRuns: true, ReadRuns: true, CancelRuns: true, Approvals: true,
		Events: true, Artifacts: true, ConfigValidation: true, Credentials: true,
	}
}

func DemoCapabilities(fixedRuns ...domain.RunID) Capabilities {
	fixed := make(map[domain.RunID]struct{}, len(fixedRuns))
	for _, runID := range fixedRuns {
		if runID != "" {
			fixed[runID] = struct{}{}
		}
	}
	return Capabilities{ListRuns: true, ReadRuns: true, Events: true, FixedRuns: fixed}
}

type Options struct {
	Application       Application
	Store             Store
	Credentials       CredentialService
	Capabilities      Capabilities
	Host              string
	MaxBodyBytes      int64
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	Secrets           []string
	Redactor          *feedback.Redactor
	Random            io.Reader
	AppShell          http.Handler
}

type Router struct {
	application  Application
	store        Store
	credentials  CredentialService
	capabilities Capabilities
	local        bool
	host         string
	maxBody      int64
	heartbeat    time.Duration
	poll         time.Duration
	random       io.Reader
	randomMu     sync.Mutex
	redactorMu   sync.RWMutex
	redactor     feedback.Redactor
	appShell     http.Handler

	bootstrapToken string
	sessionToken   string
	csrfToken      string
	bootstrapMu    sync.Mutex
	bootstrapUsed  bool
}

func NewLocal(options Options) (*Router, error) {
	if options.Host == "" || !loopbackHost(options.Host) {
		return nil, errors.New("httpapi: local host must be a literal loopback address")
	}
	router, err := newRouter(options, true)
	if err != nil {
		return nil, err
	}
	router.bootstrapToken, err = router.randomToken()
	if err != nil {
		return nil, err
	}
	router.sessionToken, err = router.randomToken()
	if err != nil {
		return nil, err
	}
	router.csrfToken, err = router.randomToken()
	if err != nil {
		return nil, err
	}
	if secureEqual(router.bootstrapToken, router.sessionToken) ||
		secureEqual(router.bootstrapToken, router.csrfToken) ||
		secureEqual(router.sessionToken, router.csrfToken) {
		return nil, errors.New("httpapi: random security tokens collided")
	}
	router.registerSecret(router.bootstrapToken)
	router.registerSecret(router.sessionToken)
	router.registerSecret(router.csrfToken)
	return router, nil
}

func NewDemo(options Options) (*Router, error) {
	fixedRuns := make([]domain.RunID, 0, len(options.Capabilities.FixedRuns))
	for runID := range options.Capabilities.FixedRuns {
		fixedRuns = append(fixedRuns, runID)
	}
	options.Capabilities = DemoCapabilities(fixedRuns...)
	options.Credentials = nil
	return newRouter(options, false)
}

func newRouter(options Options, local bool) (*Router, error) {
	if options.Application == nil || options.Store == nil {
		return nil, errors.New("httpapi: application and store are required")
	}
	if options.Capabilities.Credentials && options.Credentials == nil {
		return nil, errors.New("httpapi: credential capability requires a credential service")
	}
	bodyLimit := options.MaxBodyBytes
	if bodyLimit <= 0 {
		bodyLimit = defaultBodyLimit
	}
	heartbeat := options.HeartbeatInterval
	if heartbeat <= 0 {
		heartbeat = 15 * time.Second
	}
	poll := options.PollInterval
	if poll <= 0 {
		poll = 100 * time.Millisecond
	}
	random := options.Random
	if random == nil {
		random = cryptoRandomReader{}
	}
	redactor := feedback.NewRedactor(options.Secrets)
	if options.Redactor != nil {
		for _, secret := range options.Secrets {
			options.Redactor.Register(secret)
		}
		redactor = *options.Redactor
	}
	return &Router{
		application: options.Application, store: options.Store,
		credentials: options.Credentials, capabilities: cloneCapabilities(options.Capabilities),
		local: local, host: strings.ToLower(options.Host), maxBody: bodyLimit,
		heartbeat: heartbeat, poll: poll, random: random,
		redactor: redactor,
		appShell: options.AppShell,
	}, nil
}

func cloneCapabilities(value Capabilities) Capabilities {
	cloned := value
	if value.FixedRuns != nil {
		cloned.FixedRuns = make(map[domain.RunID]struct{}, len(value.FixedRuns))
		for runID := range value.FixedRuns {
			cloned.FixedRuns[runID] = struct{}{}
		}
	}
	return cloned
}

func (r *Router) BootstrapToken() string {
	return r.bootstrapToken
}

func (r *Router) CSRFToken() string {
	return r.csrfToken
}

func (r *Router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestID := r.requestID()
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Cache-Control", "no-store")

	if r.local && !r.validHost(request.Host) {
		r.writeError(writer, http.StatusForbidden, "INVALID_HOST", "request Host is not allowed", requestID)
		return
	}
	if request.URL.Path == "/healthz" {
		if request.Method != http.MethodGet {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if r.local && request.URL.Path == "/" && request.URL.Query().Has("bootstrap") {
		r.handleBootstrap(writer, request, requestID)
		return
	}
	if r.local {
		if !r.validSession(request) {
			r.writeError(writer, http.StatusForbidden, "SESSION_REQUIRED", "a valid local session is required", requestID)
			return
		}
		origin := request.Header.Get("Origin")
		if origin != "" && !r.validOrigin(origin) {
			r.writeError(writer, http.StatusForbidden, "INVALID_ORIGIN", "request Origin is not allowed", requestID)
			return
		}
		if mutationMethod(request.Method) {
			if origin == "" {
				r.writeError(writer, http.StatusForbidden, "INVALID_ORIGIN", "request Origin is not allowed", requestID)
				return
			}
			if !secureEqual(request.Header.Get(CSRFHeader), r.csrfToken) {
				r.writeError(writer, http.StatusForbidden, "CSRF_REQUIRED", "a valid CSRF token is required", requestID)
				return
			}
		}
	}
	if r.local && request.URL.Path == "/" {
		if request.Method != http.MethodGet {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		if r.appShell == nil {
			r.writeNotFound(writer, requestID)
			return
		}
		r.appShell.ServeHTTP(writer, request)
		return
	}
	r.dispatch(writer, request, requestID)
}

func (r *Router) dispatch(writer http.ResponseWriter, request *http.Request, requestID string) {
	parts := splitPath(request.URL.Path)
	switch {
	case len(parts) == 3 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "runs":
		switch request.Method {
		case http.MethodGet:
			if !r.capabilities.ListRuns {
				r.writeNotFound(writer, requestID)
				return
			}
			r.listRuns(writer, request, requestID)
		case http.MethodPost:
			if !r.capabilities.CreateRuns {
				r.writeNotFound(writer, requestID)
				return
			}
			r.createRun(writer, request, requestID)
		default:
			r.writeMethodNotAllowed(writer, requestID)
		}
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "runs":
		runID := domain.RunID(parts[3])
		if !r.capabilities.ReadRuns || !r.allowedRun(runID) {
			r.writeNotFound(writer, requestID)
			return
		}
		if request.Method != http.MethodGet {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.getRun(writer, request, requestID, runID)
	case len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "runs" && parts[4] == "cancel":
		runID := domain.RunID(parts[3])
		if !r.capabilities.CancelRuns || !r.allowedRun(runID) {
			r.writeNotFound(writer, requestID)
			return
		}
		if request.Method != http.MethodPost {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.cancelRun(writer, request, requestID, runID)
	case len(parts) == 7 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "runs" && parts[4] == "approvals" &&
		(parts[6] == "approve" || parts[6] == "reject"):
		runID := domain.RunID(parts[3])
		if !r.capabilities.Approvals || !r.allowedRun(runID) {
			r.writeNotFound(writer, requestID)
			return
		}
		if request.Method != http.MethodPost {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.approval(writer, request, requestID, runID, parts[5], parts[6])
	case len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "runs" && parts[4] == "events":
		runID := domain.RunID(parts[3])
		if !r.capabilities.Events || !r.allowedRun(runID) {
			r.writeNotFound(writer, requestID)
			return
		}
		if request.Method != http.MethodGet {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.events(writer, request, requestID, runID)
	case len(parts) == 6 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "runs" && parts[4] == "artifacts":
		runID := domain.RunID(parts[3])
		if !r.capabilities.Artifacts || !r.allowedRun(runID) {
			r.writeNotFound(writer, requestID)
			return
		}
		if request.Method != http.MethodGet {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.artifact(writer, request, requestID, runID, parts[5])
	case len(parts) == 4 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "config" && parts[3] == "validate":
		if !r.capabilities.ConfigValidation {
			r.writeNotFound(writer, requestID)
			return
		}
		if request.Method != http.MethodPost {
			r.writeMethodNotAllowed(writer, requestID)
			return
		}
		r.validateConfig(writer, request, requestID)
	case len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" &&
		parts[2] == "credentials":
		if !r.capabilities.Credentials {
			r.writeNotFound(writer, requestID)
			return
		}
		r.credential(writer, request, requestID, parts[3], parts[4])
	default:
		r.writeNotFound(writer, requestID)
	}
}

func (r *Router) allowedRun(runID domain.RunID) bool {
	if runID == "" {
		return false
	}
	if r.capabilities.FixedRuns == nil {
		return true
	}
	_, ok := r.capabilities.FixedRuns[runID]
	return ok
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if part == "" {
			return nil
		}
	}
	return parts
}

func mutationMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}

func loopbackHost(host string) bool {
	name := host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		name = parsed
	}
	return strings.Trim(name, "[]") == "127.0.0.1"
}

func secureEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func (r *Router) writeNotFound(writer http.ResponseWriter, requestID string) {
	r.writeError(writer, http.StatusNotFound, "NOT_FOUND", "route was not found", requestID)
}

func (r *Router) writeMethodNotAllowed(writer http.ResponseWriter, requestID string) {
	r.writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method is not allowed", requestID)
}

func (r *Router) writeJSON(writer http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		r.writeError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "request could not be completed", r.requestID())
		return
	}
	data = []byte(r.redact(string(data)))
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(append(data, '\n'))
}

func (r *Router) writeError(
	writer http.ResponseWriter,
	status int,
	code string,
	message string,
	requestID string,
) {
	message = r.redact(message)
	if len(message) > maxErrorMessage {
		message = message[:maxErrorMessage]
	}
	r.writeJSON(writer, status, map[string]any{
		"error": map[string]string{
			"code": code, "message": message, "request_id": requestID,
		},
	})
}

func (r *Router) redact(value string) string {
	r.redactorMu.RLock()
	redacted := r.redactor.Redact(value)
	r.redactorMu.RUnlock()
	return redacted
}

func (r *Router) registerSecret(secret string) {
	if secret == "" {
		return
	}
	r.redactorMu.Lock()
	r.redactor.Register(secret)
	r.redactorMu.Unlock()
}

func (r *Router) mapError(writer http.ResponseWriter, err error, requestID string) {
	switch {
	case errors.Is(err, storeport.ErrRunNotFound):
		r.writeError(writer, http.StatusNotFound, "RUN_NOT_FOUND", "run was not found", requestID)
	case errors.Is(err, storeport.ErrArtifactNotFound):
		r.writeError(writer, http.StatusNotFound, "ARTIFACT_NOT_FOUND", "artifact was not found", requestID)
	case errors.Is(err, credentials.ErrNotFound):
		r.writeError(writer, http.StatusNotFound, "CREDENTIAL_NOT_FOUND", "credential was not found", requestID)
	case errors.Is(err, credentials.ErrInvalidCredential):
		r.writeError(writer, http.StatusBadRequest, "INVALID_CREDENTIAL", "credential reference or value is invalid", requestID)
	case errors.Is(err, credentials.ErrAlreadyConfigured):
		r.writeError(writer, http.StatusConflict, "CREDENTIAL_CONFLICT", "credential is already configured", requestID)
	case errors.Is(err, credentials.ErrUnavailable), errors.Is(err, credentials.ErrLocked):
		r.writeError(writer, http.StatusServiceUnavailable, "CREDENTIAL_STORE_UNAVAILABLE", "credential store is unavailable", requestID)
	case errors.Is(err, app.ErrPreflightFailed):
		r.writeError(writer, http.StatusUnprocessableEntity, "PREFLIGHT_FAILED", "run preflight failed", requestID)
	case errors.Is(err, app.ErrRepoBusy), errors.Is(err, app.ErrRunTerminal),
		errors.Is(err, app.ErrApprovalStale):
		r.writeError(writer, http.StatusConflict, "CONFLICT", "request conflicts with current run state", requestID)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		r.writeError(writer, http.StatusRequestTimeout, "REQUEST_CANCELLED", "request was cancelled", requestID)
	default:
		r.writeError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "request could not be completed", requestID)
	}
}
