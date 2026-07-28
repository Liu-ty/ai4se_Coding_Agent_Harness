package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os/exec"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

var (
	ErrPreflightFailed = errors.New("run preflight failed")
	ErrRunTerminal     = errors.New("run is already terminal")
	ErrApprovalStale   = errors.New("approval does not match the current baseline")
)

const defaultMaxBaselineBytes int64 = 8 << 20

type CreateRunRequest struct {
	RepoRoot              string                   `json:"repo_root"`
	Task                  string                   `json:"task"`
	Provider              string                   `json:"provider"`
	Model                 string                   `json:"model"`
	Endpoint              string                   `json:"endpoint"`
	ConfirmCustomEndpoint bool                     `json:"confirm_custom_endpoint"`
	Profile               domain.PermissionProfile `json:"profile"`
	ConfigPath            string                   `json:"config_path,omitempty"`
}

type LoopController interface {
	Start(context.Context, RunSetup) error
	Approve(context.Context, domain.RunID, string) error
	Reject(context.Context, domain.RunID, string, bool) error
	Cancel(context.Context, domain.RunID) error
}

type RunSetup struct {
	Run     domain.Run
	Request CreateRunRequest
	Report  PreflightReport
	Config  config.Config
}

type CredentialStatus interface {
	Status(context.Context, credentials.Ref) (credentials.Status, error)
}

type Clock interface {
	Now() time.Time
}

type Options struct {
	Store       storeport.Store
	Loops       LoopController
	Credentials CredentialStatus
	DataDir     string
	Locks       *RepoLocks
	Clock       Clock
	NewRunID    func() domain.RunID
}

type Service struct {
	store            storeport.Store
	locks            *RepoLocks
	loops            LoopController
	creds            CredentialStatus
	dataDir          string
	clock            Clock
	newID            func() domain.RunID
	maxBaselineBytes int64

	lookPath      func(string) (string, error)
	probeWritable func(string) error

	lifecycleMu sync.Mutex
	runsMu      sync.Mutex
	cancels     map[domain.RunID]context.CancelFunc
	inputs      map[domain.RunID]runInput
	pending     map[domain.RunID]preflightApproval
	stopping    map[domain.RunID]bool
}

type runInput struct {
	request CreateRunRequest
	report  PreflightReport
	config  config.Config
}

type preflightApproval struct {
	digest string
	input  runInput
}

type PreflightError struct {
	Report PreflightReport
}

func (e *PreflightError) Error() string {
	return ErrPreflightFailed.Error()
}

func (e *PreflightError) Unwrap() error {
	return ErrPreflightFailed
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

func NewService(ctx context.Context, options Options) (*Service, error) {
	if options.Store == nil || options.Loops == nil {
		return nil, errors.New("app: store and loop controller are required")
	}
	locks := options.Locks
	if locks == nil {
		locks = NewRepoLocks()
	}
	clock := options.Clock
	if clock == nil {
		clock = systemClock{}
	}
	newID := options.NewRunID
	if newID == nil {
		newID = randomRunID
	}
	service := &Service{
		store: options.Store, locks: locks, loops: options.Loops,
		creds: options.Credentials, dataDir: options.DataDir,
		clock: clock, newID: newID, lookPath: exec.LookPath,
		maxBaselineBytes: defaultMaxBaselineBytes,
		probeWritable:    probeWritableDirectory, cancels: make(map[domain.RunID]context.CancelFunc),
		inputs: make(map[domain.RunID]runInput), pending: make(map[domain.RunID]preflightApproval),
		stopping: make(map[domain.RunID]bool),
	}
	if err := service.recoverRuns(ctx); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) CreateRun(ctx context.Context, request CreateRunRequest) (domain.Run, error) {
	result := s.preflight(ctx, request)
	if !result.report.OK {
		return domain.Run{}, &PreflightError{Report: result.report}
	}
	now := s.clock.Now().UTC()
	run := domain.Run{
		ID: s.newID(), State: domain.StateCreated, Profile: request.Profile,
		Task: request.Task, RepoRoot: result.report.RepoRoot,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.locks.Acquire(run.RepoRoot, run.ID); err != nil {
		return domain.Run{}, err
	}
	payload, err := json.Marshal(struct {
		Provider         string `json:"provider"`
		Model            string `json:"model"`
		Endpoint         string `json:"endpoint"`
		ConfigPath       string `json:"config_path,omitempty"`
		BaselineCommit   string `json:"baseline_commit"`
		BaselineDiffHash string `json:"baseline_diff_hash"`
	}{
		Provider: request.Provider, Model: request.Model, Endpoint: request.Endpoint,
		ConfigPath: request.ConfigPath, BaselineCommit: result.report.BaselineCommit,
		BaselineDiffHash: result.report.BaselineDiffHash,
	})
	if err != nil {
		s.locks.ReleaseRun(run.ID)
		return domain.Run{}, err
	}
	if _, err := s.store.CreateRunWithEvent(ctx, run, "RunCreated", payload); err != nil {
		s.locks.ReleaseRun(run.ID)
		return domain.Run{}, err
	}
	input := runInput{request: request, report: result.report, config: result.config}
	s.runsMu.Lock()
	s.inputs[run.ID] = input
	s.runsMu.Unlock()
	if hasFinding(result.report, CodeDirtyWorktreeApproval) {
		digest := preflightDigest(run, result.report)
		if err := s.transitionState(ctx, &run, domain.StatePreflight, "PreflightStarted", json.RawMessage(`{}`)); err != nil {
			return domain.Run{}, s.cleanupCreateFailure(run, err)
		}
		approvalPayload, err := json.Marshal(struct {
			Digest           string `json:"digest"`
			BaselineCommit   string `json:"baseline_commit"`
			BaselineDiffHash string `json:"baseline_diff_hash"`
		}{
			Digest: digest, BaselineCommit: result.report.BaselineCommit,
			BaselineDiffHash: result.report.BaselineDiffHash,
		})
		if err != nil {
			return domain.Run{}, s.cleanupCreateFailure(run, err)
		}
		if err := s.transitionState(
			ctx, &run, domain.StateAwaitingApproval, "PreflightApprovalRequired", approvalPayload,
		); err != nil {
			return domain.Run{}, s.cleanupCreateFailure(run, err)
		}
		s.runsMu.Lock()
		s.pending[run.ID] = preflightApproval{digest: digest, input: input}
		s.runsMu.Unlock()
		return run, nil
	}
	s.startRun(run)
	return run, nil
}

func (s *Service) startRun(run domain.Run) {
	runCtx, cancel := context.WithCancel(context.Background())
	s.runsMu.Lock()
	s.cancels[run.ID] = cancel
	s.runsMu.Unlock()
	go s.start(runCtx, run)
}

func (s *Service) GetRun(ctx context.Context, runID domain.RunID) (domain.Run, error) {
	return s.store.GetRun(ctx, runID)
}

func (s *Service) ListRuns(ctx context.Context) ([]domain.Run, error) {
	return s.store.ListRuns(ctx)
}

func (s *Service) Approve(ctx context.Context, runID domain.RunID, digest string) error {
	s.lifecycleMu.Lock()
	run, err := s.activeRun(ctx, runID)
	if err != nil {
		s.lifecycleMu.Unlock()
		return err
	}
	s.runsMu.Lock()
	pending, preflightPending := s.pending[runID]
	stopping := s.stopping[runID]
	s.runsMu.Unlock()
	s.lifecycleMu.Unlock()
	if stopping {
		return ErrRunTerminal
	}
	if preflightPending {
		if digest != pending.digest {
			return ErrApprovalStale
		}
		current := s.preflight(ctx, pending.input.request)
		if !current.report.OK ||
			current.report.BaselineCommit != pending.input.report.BaselineCommit ||
			current.report.BaselineDiffHash != pending.input.report.BaselineDiffHash ||
			preflightDigest(run, current.report) != digest {
			return ErrApprovalStale
		}
		s.lifecycleMu.Lock()
		s.runsMu.Lock()
		stopping = s.stopping[runID]
		s.runsMu.Unlock()
		if stopping {
			s.lifecycleMu.Unlock()
			return ErrRunTerminal
		}
		if err := s.transitionState(
			ctx, &run, domain.StatePreflight, "PreflightApprovalGranted", json.RawMessage(`{}`),
		); err != nil {
			s.lifecycleMu.Unlock()
			return err
		}
		s.runsMu.Lock()
		delete(s.pending, runID)
		s.inputs[runID] = runInput{
			request: pending.input.request, report: current.report, config: current.config,
		}
		s.runsMu.Unlock()
		s.lifecycleMu.Unlock()
		s.startRun(run)
		return nil
	}
	if err := s.loops.Approve(ctx, runID, digest); err != nil {
		if errors.Is(err, agent.ErrApprovalNotGranted) || errors.Is(err, ErrLoopSessionBusy) {
			return err
		}
		return s.failActiveControl(run, "LOOP_APPROVAL_FAILED", err)
	}
	return s.releaseIfTerminal(ctx, run)
}

func (s *Service) Reject(ctx context.Context, runID domain.RunID, digest string, terminate bool) error {
	s.lifecycleMu.Lock()
	run, err := s.activeRun(ctx, runID)
	if err != nil {
		s.lifecycleMu.Unlock()
		return err
	}
	s.runsMu.Lock()
	pending, preflightPending := s.pending[runID]
	stopping := s.stopping[runID]
	s.runsMu.Unlock()
	s.lifecycleMu.Unlock()
	if stopping {
		return ErrRunTerminal
	}
	if preflightPending {
		if digest != pending.digest {
			return ErrApprovalStale
		}
		if terminate {
			if err := s.stopIfActive(ctx, runID, "APPROVAL_REJECTED"); err != nil {
				return err
			}
			return s.releaseIfTerminal(ctx, run)
		}
		return nil
	}
	if err := s.loops.Reject(ctx, runID, digest, terminate); err != nil {
		if errors.Is(err, agent.ErrApprovalRejectionMismatch) || errors.Is(err, ErrLoopSessionBusy) {
			return err
		}
		return s.failActiveControl(run, "LOOP_REJECTION_FAILED", err)
	}
	if terminate {
		if err := s.stopIfActive(ctx, runID, "APPROVAL_REJECTED"); err != nil {
			return err
		}
	}
	return s.releaseIfTerminal(ctx, run)
}

func (s *Service) CancelRun(ctx context.Context, runID domain.RunID) error {
	s.lifecycleMu.Lock()
	run, err := s.activeRun(ctx, runID)
	if err != nil {
		s.lifecycleMu.Unlock()
		return err
	}
	s.runsMu.Lock()
	if s.stopping[runID] {
		s.runsMu.Unlock()
		s.lifecycleMu.Unlock()
		return ErrRunTerminal
	}
	s.stopping[runID] = true
	cancel := s.cancels[runID]
	s.runsMu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.lifecycleMu.Unlock()
	if err := s.loops.Cancel(ctx, runID); err != nil {
		s.runsMu.Lock()
		delete(s.stopping, runID)
		s.runsMu.Unlock()
		return err
	}
	if err := s.stopIfActive(context.WithoutCancel(ctx), runID, "USER_CANCELLED"); err != nil {
		return err
	}
	return s.releaseIfTerminal(context.WithoutCancel(ctx), run)
}

func (s *Service) start(ctx context.Context, run domain.Run) {
	s.runsMu.Lock()
	input, ok := s.inputs[run.ID]
	s.runsMu.Unlock()
	if !ok {
		_ = s.stopIfActive(context.Background(), run.ID, "LOOP_SETUP_MISSING")
		_ = s.releaseIfTerminal(context.Background(), run)
		return
	}
	err := s.loops.Start(ctx, RunSetup{
		Run: run, Request: input.request, Report: input.report, Config: input.config,
	})
	if err != nil {
		reason := "LOOP_FAILED"
		if errors.Is(err, context.Canceled) {
			reason = "USER_CANCELLED"
		}
		_ = s.stopIfActive(context.Background(), run.ID, reason)
	}
	_ = s.releaseIfTerminal(context.Background(), run)
}

func (s *Service) activeRun(ctx context.Context, runID domain.RunID) (domain.Run, error) {
	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return domain.Run{}, err
	}
	if terminal(run.State) {
		return domain.Run{}, ErrRunTerminal
	}
	return run, nil
}

func (s *Service) stopIfActive(ctx context.Context, runID domain.RunID, reason string) error {
	payload, err := json.Marshal(struct {
		Reason string `json:"reason"`
	}{Reason: reason})
	if err != nil {
		return err
	}
	for {
		run, err := s.store.GetRun(ctx, runID)
		if err != nil {
			return err
		}
		if terminal(run.State) {
			return nil
		}
		if err := domain.Transition(run.State, domain.StateStopped); err != nil {
			return err
		}
		expected := run.State
		run.State = domain.StateStopped
		run.UpdatedAt = s.clock.Now().UTC()
		if _, err = s.store.UpdateRunIfState(ctx, run, expected, "RunStopped", payload); errors.Is(err, storeport.ErrRunStateChanged) {
			continue
		}
		return err
	}
}

func (s *Service) releaseIfTerminal(ctx context.Context, known domain.Run) error {
	current, err := s.store.GetRun(ctx, known.ID)
	if err != nil {
		return err
	}
	if !terminal(current.State) {
		return nil
	}
	s.locks.ReleaseRun(current.ID)
	s.runsMu.Lock()
	if cancel := s.cancels[current.ID]; cancel != nil {
		cancel()
		delete(s.cancels, current.ID)
	}
	delete(s.pending, current.ID)
	delete(s.inputs, current.ID)
	delete(s.stopping, current.ID)
	s.runsMu.Unlock()
	return nil
}

func (s *Service) cleanupCreateFailure(run domain.Run, cause error) error {
	cleanupErr := s.stopIfActive(context.Background(), run.ID, "RUN_SETUP_FAILED")
	if cleanupErr == nil {
		cleanupErr = s.releaseIfTerminal(context.Background(), run)
	}
	if cleanupErr != nil {
		return errors.Join(cause, cleanupErr)
	}
	return cause
}

func (s *Service) failActiveControl(run domain.Run, reason string, cause error) error {
	cancelErr := s.loops.Cancel(context.Background(), run.ID)
	stopErr := s.stopIfActive(context.Background(), run.ID, reason)
	releaseErr := error(nil)
	if stopErr == nil {
		releaseErr = s.releaseIfTerminal(context.Background(), run)
	}
	return errors.Join(cause, cancelErr, stopErr, releaseErr)
}

func (s *Service) recoverRuns(ctx context.Context) error {
	runs, err := s.store.ListRuns(ctx)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if terminal(run.State) {
			s.locks.ReleaseRun(run.ID)
			continue
		}
		if err := domain.Transition(run.State, domain.StateStopped); err != nil {
			return err
		}
		expected := run.State
		run.State = domain.StateStopped
		run.UpdatedAt = s.clock.Now().UTC()
		payload := json.RawMessage(`{"reason":"STARTUP_RECOVERY"}`)
		if _, err := s.store.UpdateRunIfState(ctx, run, expected, "RunRecovered", payload); err != nil {
			return err
		}
		s.locks.ReleaseRun(run.ID)
	}
	return nil
}

func (s *Service) transitionState(
	ctx context.Context,
	run *domain.Run,
	next domain.RunState,
	eventType string,
	payload json.RawMessage,
) error {
	if err := domain.Transition(run.State, next); err != nil {
		return err
	}
	expected := run.State
	run.State = next
	run.UpdatedAt = s.clock.Now().UTC()
	_, err := s.store.UpdateRunIfState(ctx, *run, expected, eventType, payload)
	return err
}

func hasFinding(report PreflightReport, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func preflightDigest(run domain.Run, report PreflightReport) string {
	payload, _ := json.Marshal(struct {
		RunID            domain.RunID             `json:"run_id"`
		Profile          domain.PermissionProfile `json:"profile"`
		RepoRoot         string                   `json:"repo_root"`
		BaselineCommit   string                   `json:"baseline_commit"`
		BaselineDiffHash string                   `json:"baseline_diff_hash"`
	}{
		RunID: run.ID, Profile: run.Profile, RepoRoot: run.RepoRoot,
		BaselineCommit: report.BaselineCommit, BaselineDiffHash: report.BaselineDiffHash,
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func terminal(state domain.RunState) bool {
	return state == domain.StateSucceeded ||
		state == domain.StateReviewComplete ||
		state == domain.StateStopped
}

func randomRunID() domain.RunID {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return domain.RunID(hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano))))
	}
	return domain.RunID(hex.EncodeToString(value[:]))
}
