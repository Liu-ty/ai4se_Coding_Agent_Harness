package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/httpapi"
)

func TestLocalCompositionPublishesProductionApprovalRequest(t *testing.T) {
	root, err := prepareRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	redactor := feedback.NewRedactor(nil)
	application, target, _, err := newLocalApplication(t.TempDir(), &redactor)
	if err != nil {
		t.Fatal(err)
	}
	run, err := application.CreateRun(context.Background(), localRunRequest(root))
	if err != nil {
		t.Fatal(err)
	}
	var request agent.ApprovalRequired
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events, listErr := target.ListEvents(context.Background(), run.ID, 1)
		if listErr != nil {
			t.Fatal(listErr)
		}
		for _, event := range events {
			if event.Type == "ApprovalRequired" {
				if err := json.Unmarshal(event.Payload, &request); err != nil {
					t.Fatal(err)
				}
			}
		}
		if request.Digest != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if request.Digest == "" || request.Action.Kind != "apply_patch" ||
		len(request.AffectedFiles) != 1 ||
		request.AffectedFiles[0] != approvalCanary+".txt" ||
		request.Risk == "" || request.RiskReason == "" ||
		len(request.BaselineEvidence) != 2 {
		t.Fatalf("production approval request = %#v", request)
	}
	if err := application.Approve(
		context.Background(), run.ID, string(request.Digest),
	); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		stored, getErr := target.GetRun(context.Background(), run.ID)
		if getErr == nil && stored.State == domain.StateSucceeded {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	stored, _ := target.GetRun(context.Background(), run.ID)
	t.Fatalf("approved production loop state = %s, want %s", stored.State, domain.StateSucceeded)
}

type eventCapture struct {
	header http.Header
	body   bytes.Buffer
	cancel context.CancelFunc
}

func (w *eventCapture) Header() http.Header { return w.header }
func (*eventCapture) WriteHeader(int)       {}
func (w *eventCapture) Write(value []byte) (int, error) {
	count, err := w.body.Write(value)
	if bytes.Contains(value, []byte("ApprovalRequired")) {
		w.cancel()
	}
	return count, err
}
func (*eventCapture) Flush() {}

func TestCredentialPUTRedactsRealApprovalFromStoreAndSSE(t *testing.T) {
	const canary = "quartz-orchid-7429"
	root, err := prepareRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	redactor := feedback.NewRedactor(nil)
	application, target, credentialService, err := newLocalApplication(t.TempDir(), &redactor)
	if err != nil {
		t.Fatal(err)
	}
	api, err := httpapi.NewLocal(httpapi.Options{
		Application: application, Store: target, Credentials: credentialService,
		Capabilities: httpapi.LocalCapabilities(), Host: localAddress,
		Random: &deterministicReader{}, Redactor: &redactor,
		PollInterval: 10 * time.Millisecond, HeartbeatInterval: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := httptest.NewRequest(
		http.MethodGet, "/?bootstrap="+api.BootstrapToken(), nil,
	)
	bootstrap.Host = localAddress
	bootstrapRecorder := httptest.NewRecorder()
	api.ServeHTTP(bootstrapRecorder, bootstrap)
	cookies := bootstrapRecorder.Result().Cookies()
	if bootstrapRecorder.Code != http.StatusSeeOther || len(cookies) != 1 {
		t.Fatalf("bootstrap = %d, cookies = %#v", bootstrapRecorder.Code, cookies)
	}
	put := httptest.NewRequest(
		http.MethodPut, "/api/v1/credentials/openai/api.openai.com",
		strings.NewReader(`{"secret":"`+canary+`"}`),
	)
	put.Host = localAddress
	put.Header.Set("Origin", "http://"+localAddress)
	put.Header.Set("Content-Type", "application/json")
	put.Header.Set(httpapi.CSRFHeader, api.CSRFToken())
	put.AddCookie(cookies[0])
	putRecorder := httptest.NewRecorder()
	api.ServeHTTP(putRecorder, put)
	if putRecorder.Code != http.StatusNoContent {
		t.Fatalf("credential PUT = %d, body = %s", putRecorder.Code, putRecorder.Body.String())
	}
	run, err := application.CreateRun(context.Background(), localRunRequest(root))
	if err != nil {
		t.Fatal(err)
	}
	var approvalPayload []byte
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events, listErr := target.ListEvents(context.Background(), run.ID, 1)
		if listErr != nil {
			t.Fatal(listErr)
		}
		for _, event := range events {
			if event.Type == "ApprovalRequired" {
				approvalPayload = event.Payload
			}
		}
		if len(approvalPayload) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if bytes.Contains(approvalPayload, []byte(canary)) ||
		!bytes.Contains(approvalPayload, []byte("[REDACTED]")) {
		t.Fatalf("stored approval was not centrally redacted: %s", approvalPayload)
	}
	streamContext, cancel := context.WithCancel(context.Background())
	stream := httptest.NewRequest(
		http.MethodGet, "/api/v1/runs/"+string(run.ID)+"/events", nil,
	).WithContext(streamContext)
	stream.Host = localAddress
	stream.Header.Set("Origin", "http://"+localAddress)
	stream.AddCookie(cookies[0])
	capture := &eventCapture{header: make(http.Header), cancel: cancel}
	api.ServeHTTP(capture, stream)
	streamBody := capture.body.String()
	if strings.Contains(streamBody, canary) ||
		!strings.Contains(streamBody, "ApprovalRequired") ||
		!strings.Contains(streamBody, "[REDACTED]") {
		t.Fatalf("SSE approval was not centrally redacted: %s", streamBody)
	}
}
