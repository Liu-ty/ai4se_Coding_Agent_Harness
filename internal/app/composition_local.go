package app

import (
	"context"
	"errors"
	"sync"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

var (
	ErrLoopSessionNotFound = errors.New("run loop session not found")
	ErrLoopSessionBusy     = errors.New("run loop session is busy")
)

// AgentLoopFactory constructs one repository-scoped real loop. RunSetup carries
// the canonical repository, validated configuration, provider/model selection,
// and captured baseline needed to bind concrete adapters safely.
type AgentLoopFactory func(context.Context, RunSetup) (*agent.Loop, *policy.ApprovalStore, error)

type agentLoopController struct {
	factory  AgentLoopFactory
	redactor feedback.Redactor
	mu       sync.Mutex
	sessions map[domain.RunID]*agentLoopSession
}

type agentLoopSession struct {
	loop      *agent.Loop
	approvals *policy.ApprovalStore
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	active    chan struct{}
}

func NewAgentLoopController(factory AgentLoopFactory, redactors ...feedback.Redactor) LoopController {
	var redactor feedback.Redactor
	if len(redactors) > 0 {
		redactor = redactors[0]
	}
	return &agentLoopController{
		factory: factory, redactor: redactor, sessions: make(map[domain.RunID]*agentLoopSession),
	}
}

func (c *agentLoopController) Start(ctx context.Context, setup RunSetup) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.factory == nil {
		return errors.New("app: agent loop factory is required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	loop, approvals, err := c.factory(runCtx, setup)
	if err != nil {
		cancel()
		return err
	}
	if loop == nil || approvals == nil {
		cancel()
		return errors.New("app: loop factory returned incomplete session")
	}
	loop.BindApprovalRedactor(c.redactor)
	loop.BindBaselines(map[string]string{
		"baseline_commit":    setup.Report.BaselineCommit,
		"baseline_diff_hash": setup.Report.BaselineDiffHash,
	})
	session := &agentLoopSession{
		loop: loop, approvals: approvals, ctx: runCtx, cancel: cancel,
	}
	c.mu.Lock()
	if _, exists := c.sessions[setup.Run.ID]; exists {
		c.mu.Unlock()
		cancel()
		return errors.New("app: run loop session already exists")
	}
	c.sessions[setup.Run.ID] = session
	c.mu.Unlock()

	result, runErr := session.invoke(func(runCtx context.Context) (agent.Result, error) {
		return loop.Run(runCtx, setup.Run)
	})
	if terminal(result.State) || runErr != nil {
		c.deleteSession(setup.Run.ID, session)
	}
	return runErr
}

func (c *agentLoopController) Approve(
	ctx context.Context,
	runID domain.RunID,
	digest string,
) error {
	session, err := c.session(runID)
	if err != nil {
		return err
	}
	approvalDigest := policy.ApprovalDigest(digest)
	result, err := session.invoke(func(runCtx context.Context) (agent.Result, error) {
		session.approvals.Grant(approvalDigest)
		result, resumeErr := session.loop.ResumeApproval(runCtx, runID, approvalDigest)
		if resumeErr != nil && !terminal(result.State) {
			session.approvals.Revoke(approvalDigest)
		}
		return result, resumeErr
	})
	if terminal(result.State) || (err != nil &&
		!errors.Is(err, agent.ErrApprovalNotGranted) &&
		!errors.Is(err, ErrLoopSessionBusy)) {
		c.deleteSession(runID, session)
	}
	return err
}

func (c *agentLoopController) Reject(
	ctx context.Context,
	runID domain.RunID,
	digest string,
	terminate bool,
) error {
	session, err := c.session(runID)
	if err != nil {
		return err
	}
	approvalDigest := policy.ApprovalDigest(digest)
	result, err := session.invoke(func(runCtx context.Context) (agent.Result, error) {
		return session.loop.RejectApproval(runCtx, runID, approvalDigest, terminate)
	})
	if terminal(result.State) || (err != nil &&
		!errors.Is(err, agent.ErrApprovalRejectionMismatch) &&
		!errors.Is(err, ErrLoopSessionBusy)) {
		c.deleteSession(runID, session)
	}
	return err
}

func (c *agentLoopController) Cancel(ctx context.Context, runID domain.RunID) error {
	session, err := c.session(runID)
	if errors.Is(err, ErrLoopSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := session.cancelAndWait(ctx); err != nil {
		return err
	}
	c.deleteSession(runID, session)
	return nil
}

func (c *agentLoopController) session(runID domain.RunID) (*agentLoopSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	session := c.sessions[runID]
	if session == nil {
		return nil, ErrLoopSessionNotFound
	}
	return session, nil
}

func (c *agentLoopController) deleteSession(runID domain.RunID, session *agentLoopSession) {
	c.mu.Lock()
	if c.sessions[runID] == session {
		delete(c.sessions, runID)
	}
	c.mu.Unlock()
	session.cancel()
}

func (s *agentLoopSession) invoke(
	invoke func(context.Context) (agent.Result, error),
) (agent.Result, error) {
	s.mu.Lock()
	if err := s.ctx.Err(); err != nil {
		s.mu.Unlock()
		return agent.Result{}, err
	}
	if s.active != nil {
		s.mu.Unlock()
		return agent.Result{}, ErrLoopSessionBusy
	}
	done := make(chan struct{})
	s.active = done
	s.mu.Unlock()

	result, err := invoke(s.ctx)

	s.mu.Lock()
	if s.active == done {
		s.active = nil
		close(done)
	}
	s.mu.Unlock()
	return result, err
}

func (s *agentLoopSession) cancelAndWait(ctx context.Context) error {
	s.mu.Lock()
	s.cancel()
	active := s.active
	s.mu.Unlock()
	if active == nil {
		return nil
	}
	select {
	case <-active:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NewLocal is the local-only composition root. Public-demo composition must use
// a different constructor and cannot supply credentials or filesystem loops.
func NewLocal(
	ctx context.Context,
	store storeport.Store,
	factory AgentLoopFactory,
	creds *credentials.Service,
	dataDir string,
	redactor *feedback.Redactor,
) (*Service, error) {
	if factory == nil {
		return nil, errors.New("app: agent loop factory is required")
	}
	if redactor == nil {
		return nil, errors.New("app: central redactor is required")
	}
	return NewService(ctx, Options{
		Store: store, Loops: NewAgentLoopController(factory, *redactor),
		Credentials: creds, DataDir: dataDir,
	})
}
