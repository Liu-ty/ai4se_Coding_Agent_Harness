package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/budget"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/feedback"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/provider"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/store"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/testutil/testrepo"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

// Replacing the profile branches with one generic dirty-worktree result must
// fail this test: unattended mutation is blocked, supervised use is visibly
// gated, and review remains read-only.
func TestPreflightAppliesDirtyWorkspaceProfileRules(t *testing.T) {
	repo := configuredRepo(t, "git")
	repo.Write(t, "a.txt", "dirty\n")
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))

	tests := []struct {
		name     string
		profile  domain.PermissionProfile
		ok       bool
		code     string
		severity Severity
	}{
		{"workspace auto rejects", domain.ProfileWorkspaceAuto, false, CodeDirtyWorktree, SeverityError},
		{"supervised requires approval", domain.ProfileSupervised, true, CodeDirtyWorktreeApproval, SeverityWarning},
		{"review records read only dirt", domain.ProfileReview, true, CodeDirtyWorktreeReadOnly, SeverityInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := svc.Preflight(context.Background(), request(repo.Root, tt.profile))
			if report.OK != tt.ok {
				t.Fatalf("OK = %v, findings = %#v", report.OK, report.Findings)
			}
			finding := findingByCode(t, report, tt.code)
			if finding.Severity != tt.severity {
				t.Fatalf("severity = %q, want %q", finding.Severity, tt.severity)
			}
			if report.BaselineCommit == "" || report.BaselineDiffHash == "" {
				t.Fatalf("baseline is incomplete: %#v", report)
			}
		})
	}
}

// Collapsing environment failures into a generic "preflight failed" code must
// fail this test because callers need a deterministic next action.
func TestPreflightReturnsDistinctFindingCodes(t *testing.T) {
	t.Run("git missing", func(t *testing.T) {
		svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
		svc.lookPath = func(name string) (string, error) {
			if name == "git" {
				return "", exec.ErrNotFound
			}
			return exec.LookPath(name)
		}
		report := svc.Preflight(context.Background(), request(t.TempDir(), domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeGitMissing)
	})

	t.Run("not repository", func(t *testing.T) {
		svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
		report := svc.Preflight(context.Background(), request(t.TempDir(), domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeNotGitRepository)
	})

	t.Run("invalid configuration", func(t *testing.T) {
		repo := testrepo.New(t, map[string]string{".ai4se-harness.toml": "version = 99\n"})
		svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
		report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeInvalidConfig)
	})

	t.Run("missing required executable", func(t *testing.T) {
		repo := configuredRepo(t, "ai4se-command-that-does-not-exist")
		svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
		report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeExecutableMissing)
	})

	t.Run("credential not configured", func(t *testing.T) {
		repo := configuredRepo(t, "git")
		creds := credentials.NewService(credentials.NewMemoryStore(), nil)
		svc := newTestService(t, store.NewMemory(), nopController{}, creds)
		report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeCredentialMissing)
	})

	t.Run("credential store unavailable", func(t *testing.T) {
		repo := configuredRepo(t, "git")
		svc := newTestService(t, store.NewMemory(), nopController{}, statusStub{err: credentials.ErrUnavailable})
		report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeCredentialStoreUnavailable)
	})

	t.Run("credential endpoint mismatch", func(t *testing.T) {
		repo := configuredRepo(t, "git")
		svc := newTestService(t, store.NewMemory(), nopController{}, statusStub{status: credentials.Status{
			Configured: true,
			Ref:        credentials.Ref{Provider: "openai", Host: "other.example"},
		}})
		report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeCredentialEndpointMismatch)
	})

	t.Run("data directory unavailable", func(t *testing.T) {
		repo := configuredRepo(t, "git")
		svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
		svc.probeWritable = func(string) error { return errors.New("read only") }
		report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
		findingByCode(t, report, CodeDataDirectoryUnavailable)
	})
}

func TestEndpointQuerySecretIsRejectedAndNeverPersisted(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	svc := newTestService(t, st, nopController{}, configuredCredentials(t))
	req := request(repo.Root, domain.ProfileWorkspaceAuto)
	req.Endpoint = "https://api.openai.com/v1?api_key=query-canary-secret"

	report := svc.Preflight(context.Background(), req)
	finding := findingByCode(t, report, CodeInvalidEndpoint)
	if containsFold(finding.Message, "query-canary-secret") {
		t.Fatalf("finding leaked endpoint secret: %#v", finding)
	}
	if _, err := svc.CreateRun(context.Background(), req); !errors.Is(err, ErrPreflightFailed) {
		t.Fatalf("create error = %v, want ErrPreflightFailed", err)
	}
	runs, err := st.ListRuns(context.Background())
	if err != nil || len(runs) != 0 {
		t.Fatalf("persisted runs = %#v, %v", runs, err)
	}
}

func TestPreflightRejectsPublicHTTPAndRequiresCustomEndpointConfirmation(t *testing.T) {
	repo := configuredRepo(t, "git")
	svc := newTestService(t, store.NewMemory(), nopController{}, statusStub{status: credentials.Status{
		Configured: true, Ref: credentials.Ref{Provider: "openai", Host: "gateway.example.test"},
	}})
	publicHTTP := request(repo.Root, domain.ProfileReview)
	publicHTTP.Endpoint = "http://gateway.example.test"
	findingByCode(t, svc.Preflight(context.Background(), publicHTTP), CodeInvalidEndpoint)

	custom := request(repo.Root, domain.ProfileReview)
	custom.Endpoint = "https://gateway.example.test"
	findingByCode(t, svc.Preflight(context.Background(), custom), CodeEndpointConfirmationRequired)
	custom.ConfirmCustomEndpoint = true
	if report := svc.Preflight(context.Background(), custom); !report.OK {
		t.Fatalf("confirmed custom endpoint report = %#v", report)
	}
}

func TestRepoLocksCoordinateAcrossInstancesAndOnlyReleaseMatchingLease(t *testing.T) {
	repo := configuredRepo(t, "git")
	lockDir := t.TempDir()
	first := NewRepoLocksAt(lockDir)
	second := NewRepoLocksAt(lockDir)
	if err := first.Acquire(repo.Root, "run-a"); err != nil {
		t.Fatal(err)
	}
	if err := second.Acquire(repo.Root, "run-b"); !errors.Is(err, ErrRepoBusy) {
		t.Fatalf("cross-instance acquire error = %v, want ErrRepoBusy", err)
	}
	second.Release(repo.Root, "run-b")
	if err := second.Acquire(repo.Root, "run-b"); !errors.Is(err, ErrRepoBusy) {
		t.Fatalf("mismatched lease release removed run-a lock: %v", err)
	}
	first.Release(repo.Root, "run-a")
	if err := second.Acquire(repo.Root, "run-b"); err != nil {
		t.Fatalf("acquire after matching lease release = %v", err)
	}
}

// Hashing only porcelain status must fail this test: approval baselines need to
// distinguish different bytes at the same dirty path.
func TestPreflightBaselineDiffHashChangesWithWorkspaceContent(t *testing.T) {
	repo := configuredRepo(t, "git")
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
	repo.Write(t, "a.txt", "first edit\n")
	first := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))
	repo.Write(t, "a.txt", "second edit\n")
	second := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))

	if first.BaselineDiffHash == "" || second.BaselineDiffHash == "" {
		t.Fatalf("empty baseline hashes: %q, %q", first.BaselineDiffHash, second.BaselineDiffHash)
	}
	if first.BaselineDiffHash == second.BaselineDiffHash {
		t.Fatalf("different workspace content produced one baseline hash %q", first.BaselineDiffHash)
	}
}

func TestPreflightBaselineDiffHashChangesWithUntrackedContent(t *testing.T) {
	repo := configuredRepo(t, "git")
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
	repo.Write(t, "new.txt", "first untracked edit\n")
	first := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))
	repo.Write(t, "new.txt", "second untracked edit\n")
	second := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))

	if first.BaselineDiffHash == second.BaselineDiffHash {
		t.Fatalf("different untracked content produced one baseline hash %q", first.BaselineDiffHash)
	}
}

func TestPreflightRejectsOversizedUntrackedFile(t *testing.T) {
	repo := configuredRepo(t, "git")
	repo.Write(t, "large-untracked.txt", strings.Repeat("x", 256))
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
	svc.maxBaselineBytes = 128

	report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))

	findingByCode(t, report, CodeBaselineTooLarge)
	if report.OK {
		t.Fatalf("oversized untracked baseline reported OK: %#v", report)
	}
}

func TestPreflightRejectsUntrackedSymlink(t *testing.T) {
	repo := configuredRepo(t, "git")
	target := filepath.Join(t.TempDir(), "credential.txt")
	if err := os.WriteFile(target, []byte("outside-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(repo.Root, "untracked-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create file symlink: %v", err)
	}
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))

	report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))

	findingByCode(t, report, CodeBaselineUnavailable)
	if report.OK {
		t.Fatalf("untracked symlink baseline reported OK: %#v", report)
	}
}

// Omitting the append-only creation event must fail this test: downstream
// orchestration and audit consumers need the selected non-secret run inputs
// and captured baseline even though domain.Run stays provider-neutral.
func TestCreateRunAppendsInitialEventWithoutCredentialMaterial(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	svc := newTestService(t, st, nopController{}, configuredCredentials(t))
	runRequest := request(repo.Root, domain.ProfileWorkspaceAuto)
	run, err := svc.CreateRun(context.Background(), runRequest)
	if err != nil {
		t.Fatal(err)
	}
	events, err := st.ListEvents(context.Background(), run.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "RunCreated" {
		t.Fatalf("events = %#v, want one RunCreated event", events)
	}
	var payload struct {
		Provider         string `json:"provider"`
		Model            string `json:"model"`
		Endpoint         string `json:"endpoint"`
		BaselineCommit   string `json:"baseline_commit"`
		BaselineDiffHash string `json:"baseline_diff_hash"`
	}
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Provider != runRequest.Provider || payload.Model != runRequest.Model ||
		payload.Endpoint != runRequest.Endpoint || payload.BaselineCommit == "" ||
		payload.BaselineDiffHash == "" {
		t.Fatalf("creation payload = %#v", payload)
	}
	if string(events[0].Payload) == "" ||
		containsFold(string(events[0].Payload), "test-only-key") ||
		containsFold(string(events[0].Payload), "authorization") {
		t.Fatalf("creation payload contains credential material: %s", events[0].Payload)
	}
}

func TestDirtySupervisedRunRequiresExactCurrentBaselineApproval(t *testing.T) {
	t.Run("exact unchanged baseline starts", func(t *testing.T) {
		repo := configuredRepo(t, "git")
		repo.Write(t, "a.txt", "dirty supervised work\n")
		controller := &startCountingController{}
		st := store.NewMemory()
		svc := newTestService(t, st, controller, configuredCredentials(t))
		run, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileSupervised))
		if err != nil {
			t.Fatal(err)
		}
		if run.State != domain.StateAwaitingApproval || controller.starts() != 0 {
			t.Fatalf("run/starts = %#v/%d", run, controller.starts())
		}
		digest := preflightApprovalDigestFromEvents(t, st, run.ID)
		if err := svc.Approve(context.Background(), run.ID, digest); err != nil {
			t.Fatal(err)
		}
		eventually(t, func() bool { return controller.starts() == 1 })
	})

	t.Run("changed baseline is stale", func(t *testing.T) {
		repo := configuredRepo(t, "git")
		repo.Write(t, "a.txt", "first dirty state\n")
		controller := &startCountingController{}
		st := store.NewMemory()
		svc := newTestService(t, st, controller, configuredCredentials(t))
		run, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileSupervised))
		if err != nil {
			t.Fatal(err)
		}
		digest := preflightApprovalDigestFromEvents(t, st, run.ID)
		repo.Write(t, "a.txt", "different dirty state\n")
		if err := svc.Approve(context.Background(), run.ID, digest); !errors.Is(err, ErrApprovalStale) {
			t.Fatalf("approval error = %v, want ErrApprovalStale", err)
		}
		if controller.starts() != 0 {
			t.Fatalf("stale approval started loop %d times", controller.starts())
		}
		stored, err := st.GetRun(context.Background(), run.ID)
		if err != nil || stored.State != domain.StateAwaitingApproval {
			t.Fatalf("stored run = %#v, %v", stored, err)
		}
	})
}

func TestCancelSerializesAgainstApproveAndKeepsLockUntilTerminal(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	controller := newCancelRaceController(st)
	svc := newTestService(t, st, controller, configuredCredentials(t))
	run, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
	if err != nil {
		t.Fatal(err)
	}

	cancelDone := make(chan error, 1)
	go func() { cancelDone <- svc.CancelRun(context.Background(), run.ID) }()
	<-controller.cancelEntered
	approveDone := make(chan error, 1)
	go func() { approveDone <- svc.Approve(context.Background(), run.ID, "digest") }()
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto)); !errors.Is(err, ErrRepoBusy) {
		t.Fatalf("repository unlocked before cancellation completed: %v", err)
	}
	close(controller.allowCancel)
	if err := <-cancelDone; err != nil {
		t.Fatal(err)
	}
	if err := <-approveDone; !errors.Is(err, ErrRunTerminal) {
		t.Fatalf("approve-after-cancel error = %v, want ErrRunTerminal", err)
	}
	if controller.approves() != 0 {
		t.Fatalf("approval executed %d times after cancellation", controller.approves())
	}
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto)); err != nil {
		t.Fatalf("repository remained locked after terminal cancellation: %v", err)
	}
}

func TestPreflightRejectsBaselineBeyondConfiguredBound(t *testing.T) {
	repo := configuredRepo(t, "git")
	repo.Write(t, "a.txt", strings.Repeat("changed bytes\n", 32))
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))
	svc.maxBaselineBytes = 64
	report := svc.Preflight(context.Background(), request(repo.Root, domain.ProfileReview))
	findingByCode(t, report, CodeBaselineTooLarge)
	if report.OK {
		t.Fatalf("oversized baseline reported OK: %#v", report)
	}
}

// Removing EvalSymlinks, Git-root normalization, or atomic acquisition must
// fail this test by admitting more than one run for the same repository.
func TestOnlyOneActiveRunPerCanonicalRepositoryUnderConcurrency(t *testing.T) {
	repo := configuredRepo(t, "git")
	link := directoryLink(t, repo.Root)
	controller := newBlockingController()
	svc := newTestService(t, store.NewMemory(), controller, configuredCredentials(t))

	const contenders = 12
	paths := make([]string, contenders)
	for i := range paths {
		if i%2 == 0 {
			paths[i] = repo.Root
		} else {
			paths[i] = link
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, contenders)
	for _, root := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CreateRun(context.Background(), request(root, domain.ProfileWorkspaceAuto))
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	successes, busy := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrRepoBusy):
			busy++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if successes != 1 || busy != contenders-1 {
		t.Fatalf("successes/busy = %d/%d, want 1/%d", successes, busy, contenders-1)
	}
	close(controller.release)
}

// Removing delegation or terminal lock release from any control operation must
// fail one of these cases.
func TestApprovalRejectAndCancelReleaseTerminalRepositoryLocks(t *testing.T) {
	tests := []struct {
		name string
		act  func(context.Context, *Service, domain.RunID) error
		want string
	}{
		{"approve", func(ctx context.Context, svc *Service, id domain.RunID) error {
			return svc.Approve(ctx, id, "digest")
		}, "approve"},
		{"reject terminal", func(ctx context.Context, svc *Service, id domain.RunID) error {
			return svc.Reject(ctx, id, "digest", true)
		}, "reject"},
		{"cancel", func(ctx context.Context, svc *Service, id domain.RunID) error {
			return svc.CancelRun(ctx, id)
		}, "cancel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := configuredRepo(t, "git")
			st := store.NewMemory()
			controller := &terminalController{store: st}
			svc := newTestService(t, st, controller, configuredCredentials(t))
			run, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
			if err != nil {
				t.Fatal(err)
			}
			if err := tt.act(context.Background(), svc, run.ID); err != nil {
				t.Fatal(err)
			}
			if controller.lastOperation() != tt.want {
				t.Fatalf("operation = %q, want %q", controller.lastOperation(), tt.want)
			}
			if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto)); err != nil {
				t.Fatalf("repository lock was not released: %v", err)
			}
		})
	}
}

// Releasing after Start merely returns, instead of after observing a terminal
// snapshot, must fail the first assertion in this test.
func TestRepositoryLockTracksTerminalState(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	controller := &stateController{store: st, state: domain.StateAwaitingApproval}
	svc := newTestService(t, st, controller, configuredCredentials(t))
	run, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto))
	if err != nil {
		t.Fatal(err)
	}
	if !waitCondition(func() bool {
		stored, getErr := st.GetRun(context.Background(), run.ID)
		return getErr == nil && stored.State == domain.StateAwaitingApproval
	}) {
		stored, _ := st.GetRun(context.Background(), run.ID)
		events, _ := st.ListEvents(context.Background(), run.ID, 1)
		t.Fatalf("real loop did not await approval: stored=%#v events=%#v", stored, events)
	}
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto)); !errors.Is(err, ErrRepoBusy) {
		t.Fatalf("create while awaiting approval error = %v, want ErrRepoBusy", err)
	}

	controller.mu.Lock()
	controller.state = domain.StateSucceeded
	controller.mu.Unlock()
	if err := svc.Approve(context.Background(), run.ID, "digest"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto)); err != nil {
		t.Fatalf("create after terminal state: %v", err)
	}
}

// Removing startup recovery must fail because the abandoned snapshot would
// remain active and could strand repository ownership across restarts.
func TestStartupRecoveryStopsAbandonedRunsAndLeavesRepositoryUnlocked(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	abandoned := domain.Run{
		ID: "abandoned", State: domain.StateDeciding, Profile: domain.ProfileWorkspaceAuto,
		Task: "old", RepoRoot: repo.Root, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0),
	}
	if err := st.CreateRun(context.Background(), abandoned); err != nil {
		t.Fatal(err)
	}
	svc := newTestService(t, st, nopController{}, configuredCredentials(t))

	recovered, err := st.GetRun(context.Background(), abandoned.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.State != domain.StateStopped {
		t.Fatalf("state = %s, want %s", recovered.State, domain.StateStopped)
	}
	events, err := st.ListEvents(context.Background(), abandoned.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "RunRecovered" ||
		!json.Valid(events[0].Payload) || string(events[0].Payload) != `{"reason":"STARTUP_RECOVERY"}` {
		t.Fatalf("recovery events = %#v", events)
	}
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileWorkspaceAuto)); err != nil {
		t.Fatalf("create after recovery: %v", err)
	}
}

func TestStartupRecoveryReleasesInjectedLockAfterRepositoryMoves(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	abandoned := domain.Run{
		ID: "abandoned-moved", State: domain.StateDeciding, Profile: domain.ProfileWorkspaceAuto,
		Task: "old", RepoRoot: repo.Root,
	}
	if err := st.CreateRun(context.Background(), abandoned); err != nil {
		t.Fatal(err)
	}
	locks := NewRepoLocks()
	if err := locks.Acquire(repo.Root, abandoned.ID); err != nil {
		t.Fatal(err)
	}
	moved := repo.Root + "-moved"
	if err := os.Rename(repo.Root, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repo.Root, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := NewService(context.Background(), Options{
		Store: st, Loops: nopController{}, Credentials: configuredCredentials(t),
		DataDir: t.TempDir(), Locks: locks,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := locks.Acquire(repo.Root, "replacement"); err != nil {
		t.Fatalf("recovered run retained moved repository lock: %v", err)
	}
}

func TestStartupRecoveryRemovesMatchingDurableLeaseAfterRepositoryMoves(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	abandoned := domain.Run{ID: "durable-moved", State: domain.StateDeciding, Profile: domain.ProfileWorkspaceAuto, RepoRoot: repo.Root}
	if err := st.CreateRun(context.Background(), abandoned); err != nil {
		t.Fatal(err)
	}
	lockDir := t.TempDir()
	locks := NewRepoLocksAt(lockDir)
	if err := locks.Acquire(repo.Root, abandoned.ID); err != nil {
		t.Fatal(err)
	}
	moved := repo.Root + "-moved"
	if err := os.Rename(repo.Root, moved); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(context.Background(), Options{
		Store: st, Loops: nopController{}, Credentials: configuredCredentials(t), DataDir: t.TempDir(), Locks: NewRepoLocksAt(lockDir),
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(lockDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".lease") {
			t.Fatalf("recovery retained durable lease %q", entry.Name())
		}
	}
}

func TestNewLocalBindsRunInputsToRealAgentLoopAndApproval(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	creds := configuredCredentials(t)
	patch := domain.Action{
		Kind: "apply_patch",
		Args: json.RawMessage(`{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`),
	}
	var decisions int
	mock := provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
		decisions++
		if decisions == 1 {
			return provider.Response{Decision: domain.AgentDecision{
				Version: "1", Action: patch,
			}}, nil
		}
		return provider.Response{Decision: domain.AgentDecision{
			Version: "1", Action: domain.Action{Kind: "finish", Args: json.RawMessage(`{}`)},
		}}, nil
	})
	runner := &realLoopRunner{}
	var received RunSetup
	var approvalStore *policy.ApprovalStore
	factory := AgentLoopFactory(func(_ context.Context, setup RunSetup) (*agent.Loop, *policy.ApprovalStore, error) {
		received = setup
		approvals := policy.NewApprovalStore()
		approvalStore = approvals
		loop := agent.New(agent.Dependencies{
			Store: st, Provider: mock, Actions: runner, Policy: policy.NewEngine(),
			Approvals: approvals, Feedback: feedback.Pipeline{},
			Validation: &realLoopChecks{results: []agent.ValidationResult{
				{StageID: "unit", Passed: true},
				{StageID: "full", Passed: true},
			}},
			Budget: budget.New(budget.Limits{
				MaxDecisions: 4, MaxMutations: 2, MaxProtocolRepairs: 1, WallClock: time.Minute,
			}, appFixedClock{}),
		})
		return loop, approvals, nil
	})
	svc, err := NewLocal(context.Background(), st, factory, creds, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	req := request(repo.Root, domain.ProfileSupervised)
	run, err := svc.CreateRun(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !waitCondition(func() bool {
		stored, getErr := st.GetRun(context.Background(), run.ID)
		return getErr == nil && stored.State == domain.StateAwaitingApproval
	}) {
		stored, _ := st.GetRun(context.Background(), run.ID)
		events, _ := st.ListEvents(context.Background(), run.ID, 1)
		t.Fatalf("real loop did not await approval: stored=%#v events=%#v", stored, events)
	}
	expectedRoot, err := workspace.ResolveRoot(repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	if received.Run.ID != run.ID || received.Request.Model != req.Model ||
		received.Report.RepoRoot != expectedRoot || len(received.Config.Validation) != 1 {
		t.Fatalf("factory setup = %#v", received)
	}
	digest := policy.Digest(run.ID, run.Profile, patch, map[string]string{
		"baseline_commit":    received.Report.BaselineCommit,
		"baseline_diff_hash": received.Report.BaselineDiffHash,
	})
	if err := svc.Approve(context.Background(), run.ID, "wrong-digest"); err == nil {
		t.Fatal("wrong digest was accepted")
	}
	if approvalStore == nil || approvalStore.Consume("wrong-digest") {
		t.Fatal("failed nonterminal approval left a consumable grant")
	}
	if err := svc.Approve(context.Background(), run.ID, string(digest)); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		stored, getErr := st.GetRun(context.Background(), run.ID)
		return getErr == nil && stored.State == domain.StateSucceeded
	})
	if runner.calls != 1 {
		t.Fatalf("real loop action calls = %d, want 1", runner.calls)
	}
}

func TestApprovedActionCanBeCancelledWithoutBlockingAnotherRun(t *testing.T) {
	repoA := configuredRepo(t, "git")
	repoB := configuredRepo(t, "git")
	st := store.NewMemory()
	actionStarted := make(chan struct{})
	runner := &cancelAwareRunner{started: actionStarted}
	factory := approvalLoopFactory(st, runner)
	svc, err := NewLocal(context.Background(), st, factory, configuredCredentials(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runA, digestA := createAwaitingRun(t, svc, st, repoA.Root)
	runB, _ := createAwaitingRun(t, svc, st, repoB.Root)

	approveDone := make(chan error, 1)
	go func() { approveDone <- svc.Approve(context.Background(), runA.ID, digestA) }()
	<-actionStarted
	if err := svc.Approve(context.Background(), runA.ID, digestA); !errors.Is(err, ErrLoopSessionBusy) {
		t.Fatalf("concurrent approval error = %v, want ErrLoopSessionBusy", err)
	}
	select {
	case err := <-approveDone:
		t.Fatalf("concurrent approval cancelled active approval: %v", err)
	default:
	}

	cancelBDone := make(chan error, 1)
	go func() { cancelBDone <- svc.CancelRun(context.Background(), runB.ID) }()
	select {
	case err := <-cancelBDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("run B cancellation was blocked by run A approval")
	}

	if err := svc.CancelRun(context.Background(), runA.ID); err != nil {
		t.Fatal(err)
	}
	if err := <-approveDone; err == nil {
		t.Fatal("cancelled approval unexpectedly succeeded")
	}
	stored, err := st.GetRun(context.Background(), runA.ID)
	if err != nil || stored.State != domain.StateStopped {
		t.Fatalf("cancelled approved run = %#v, %v", stored, err)
	}
}

func TestBusyApprovalDoesNotRevokeExistingSameDigestGrant(t *testing.T) {
	runID := domain.RunID("run-busy-approval")
	digest := policy.ApprovalDigest("same-digest")
	approvals := policy.NewApprovalStore()
	approvals.Grant(digest)
	session := &agentLoopSession{
		approvals: approvals,
		ctx:       context.Background(),
		active:    make(chan struct{}),
	}
	controller := &agentLoopController{
		sessions: map[domain.RunID]*agentLoopSession{runID: session},
	}

	if err := controller.Approve(context.Background(), runID, string(digest)); !errors.Is(err, ErrLoopSessionBusy) {
		t.Fatalf("approve error = %v, want ErrLoopSessionBusy", err)
	}
	if !approvals.Consume(digest) {
		t.Fatal("busy approval revoked an existing same-digest grant")
	}
}

func TestNewLocalRejectsNilLoopFactory(t *testing.T) {
	_, err := NewLocal(
		context.Background(), store.NewMemory(), nil, configuredCredentials(t), t.TempDir(),
	)
	if err == nil {
		t.Fatal("nil loop factory was accepted")
	}
}

func TestApprovedActionErrorStopsRunAndReleasesRepository(t *testing.T) {
	repo := configuredRepo(t, "git")
	st := store.NewMemory()
	factory := approvalLoopFactory(st, errorRunner{err: errors.New("action failed")})
	svc, err := NewLocal(context.Background(), st, factory, configuredCredentials(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run, digest := createAwaitingRun(t, svc, st, repo.Root)

	if err := svc.Approve(context.Background(), run.ID, digest); err == nil {
		t.Fatal("action failure was not returned")
	}
	stored, err := st.GetRun(context.Background(), run.ID)
	if err != nil || stored.State != domain.StateStopped {
		t.Fatalf("failed approved run = %#v, %v", stored, err)
	}
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileReview)); err != nil {
		t.Fatalf("repository lock retained after action failure: %v", err)
	}
}

func TestCreateRunSetupFailureStopsDurableRunBeforeReleasingLock(t *testing.T) {
	repo := configuredRepo(t, "git")
	base := store.NewMemory()
	st := &failConditionalUpdateStore{Store: base, remaining: 1}
	svc := newTestService(t, st, nopController{}, configuredCredentials(t))
	repo.Write(t, "a.txt", "dirty\n")

	run, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileSupervised))
	if err == nil {
		t.Fatal("injected setup failure was not returned")
	}
	runs, listErr := base.ListRuns(context.Background())
	if listErr != nil || len(runs) != 1 || runs[0].State != domain.StateStopped {
		t.Fatalf("durable runs after setup failure = %#v, %v (returned %#v)", runs, listErr, run)
	}
	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileReview)); err != nil {
		t.Fatalf("repository lock retained after durable cleanup: %v", err)
	}
}

// Any accidental reset, clean, checkout, or file rewrite in preflight/create
// must fail this test.
func TestServiceNeverChangesUserWorkspaceContent(t *testing.T) {
	repo := configuredRepo(t, "git")
	repo.Write(t, "a.txt", "user work must remain\n")
	before := gitStatus(t, repo.Root)
	svc := newTestService(t, store.NewMemory(), nopController{}, configuredCredentials(t))

	if _, err := svc.CreateRun(context.Background(), request(repo.Root, domain.ProfileReview)); err != nil {
		t.Fatal(err)
	}
	if got := repo.Read(t, "a.txt"); got != "user work must remain\n" {
		t.Fatalf("workspace content changed to %q", got)
	}
	if after := gitStatus(t, repo.Root); after != before {
		t.Fatalf("workspace status changed:\nbefore %q\nafter  %q", before, after)
	}
}

func newTestService(t *testing.T, st storeport.Store, loops LoopController, creds CredentialStatus) *Service {
	t.Helper()
	svc, err := NewService(context.Background(), Options{
		Store:       st,
		Loops:       loops,
		Credentials: creds,
		DataDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func configuredRepo(t *testing.T, executable string) *testrepo.Repo {
	t.Helper()
	configText := fmt.Sprintf(`version = 1
default_profile = "workspace-auto"

[[validation]]
id = "unit"
kind = "targeted-test"
executable = %q
args = ["--version"]
working_directory = "."
timeout = "30s"
max_output_bytes = 4096
required = true
`, executable)
	return testrepo.New(t, map[string]string{
		".ai4se-harness.toml": configText,
		"a.txt":               "old\n",
	})
}

func configuredCredentials(t *testing.T) *credentials.Service {
	t.Helper()
	creds := credentials.NewService(credentials.NewMemoryStore(), nil)
	if err := creds.Add(context.Background(), credentials.Ref{
		Provider: "openai",
		Host:     "api.openai.com",
	}, []byte("test-only-key")); err != nil {
		t.Fatal(err)
	}
	return creds
}

func request(root string, profile domain.PermissionProfile) CreateRunRequest {
	return CreateRunRequest{
		RepoRoot: root,
		Task:     "repair the test",
		Provider: "openai",
		Model:    "test-model",
		Endpoint: "https://api.openai.com",
		Profile:  profile,
	}
}

func findingByCode(t *testing.T, report PreflightReport, code string) Finding {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			return finding
		}
	}
	t.Fatalf("finding %q absent from %#v", code, report.Findings)
	return Finding{}
}

func preflightApprovalDigestFromEvents(t *testing.T, st storeport.Store, runID domain.RunID) string {
	t.Helper()
	events, err := st.ListEvents(context.Background(), runID, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type != "PreflightApprovalRequired" {
			continue
		}
		var payload struct {
			Digest string `json:"digest"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Digest == "" {
			t.Fatal("preflight approval digest is empty")
		}
		return payload.Digest
	}
	t.Fatalf("PreflightApprovalRequired absent from %#v", events)
	return ""
}

func directoryLink(t *testing.T, target string) string {
	t.Helper()
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(target, link); err == nil {
		return link
	}
	if runtime.GOOS == "windows" {
		command := exec.Command("cmd.exe", "/c", "mklink", "/J", link, target)
		if output, err := command.CombinedOutput(); err == nil {
			return link
		} else {
			t.Skipf("cannot create directory link: %v: %s", err, output)
		}
	}
	t.Skip("cannot create directory symlink")
	return ""
}

func gitStatus(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("git", "status", "--porcelain=v1")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func eventually(t *testing.T, condition func() bool) {
	t.Helper()
	if waitCondition(condition) {
		return
	}
	t.Fatal("condition was not met")
}

func waitCondition(condition func() bool) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func containsFold(value, needle string) bool {
	value = strings.ToLower(value)
	needle = strings.ToLower(needle)
	return strings.Contains(value, needle)
}

type statusStub struct {
	status credentials.Status
	err    error
}

func (s statusStub) Status(context.Context, credentials.Ref) (credentials.Status, error) {
	return s.status, s.err
}

type appFixedClock struct{}

func (appFixedClock) Now() time.Time { return time.Unix(0, 0) }

type realLoopRunner struct {
	mu    sync.Mutex
	calls int
}

func (r *realLoopRunner) Execute(context.Context, domain.Action) (agent.ActionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return agent.ActionResult{DiffDigest: "changed"}, nil
}

type cancelAwareRunner struct {
	started chan struct{}
	once    sync.Once
}

func (r *cancelAwareRunner) Execute(ctx context.Context, _ domain.Action) (agent.ActionResult, error) {
	r.once.Do(func() { close(r.started) })
	<-ctx.Done()
	return agent.ActionResult{}, ctx.Err()
}

type errorRunner struct{ err error }

func (r errorRunner) Execute(context.Context, domain.Action) (agent.ActionResult, error) {
	return agent.ActionResult{}, r.err
}

func approvalLoopFactory(st storeport.Store, actions agent.ActionExecutor) AgentLoopFactory {
	return func(_ context.Context, setup RunSetup) (*agent.Loop, *policy.ApprovalStore, error) {
		patch := domain.Action{
			Kind: "apply_patch",
			Args: json.RawMessage(`{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`),
		}
		mock := provider.NewMock(func(context.Context, provider.Request) (provider.Response, error) {
			return provider.Response{Decision: domain.AgentDecision{Version: "1", Action: patch}}, nil
		})
		approvals := policy.NewApprovalStore()
		return agent.New(agent.Dependencies{
			Store: st, Provider: mock, Actions: actions, Policy: policy.NewEngine(),
			Approvals: approvals, Feedback: feedback.Pipeline{}, Validation: &realLoopChecks{},
			Budget: budget.New(budget.Limits{
				MaxDecisions: 2, MaxMutations: 1, MaxProtocolRepairs: 1, WallClock: time.Minute,
			}, appFixedClock{}),
		}), approvals, nil
	}
}

func createAwaitingRun(
	t *testing.T,
	svc *Service,
	st storeport.Store,
	root string,
) (domain.Run, string) {
	t.Helper()
	run, err := svc.CreateRun(context.Background(), request(root, domain.ProfileSupervised))
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		stored, getErr := st.GetRun(context.Background(), run.ID)
		return getErr == nil && stored.State == domain.StateAwaitingApproval
	})
	patch := domain.Action{
		Kind: "apply_patch",
		Args: json.RawMessage(`{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n"}`),
	}
	report := svc.Preflight(context.Background(), request(root, domain.ProfileSupervised))
	if !report.OK {
		t.Fatalf("baseline used for approval digest is unavailable: %#v", report)
	}
	return run, string(policy.Digest(run.ID, run.Profile, patch, map[string]string{
		"baseline_commit":    report.BaselineCommit,
		"baseline_diff_hash": report.BaselineDiffHash,
	}))
}

type failConditionalUpdateStore struct {
	storeport.Store
	mu        sync.Mutex
	remaining int
}

func (s *failConditionalUpdateStore) UpdateRunIfState(
	ctx context.Context,
	run domain.Run,
	expected domain.RunState,
	eventType string,
	payload json.RawMessage,
) (domain.RunEvent, error) {
	s.mu.Lock()
	if s.remaining > 0 {
		s.remaining--
		s.mu.Unlock()
		return domain.RunEvent{}, errors.New("injected conditional update failure")
	}
	s.mu.Unlock()
	return s.Store.UpdateRunIfState(ctx, run, expected, eventType, payload)
}

type realLoopChecks struct {
	results []agent.ValidationResult
}

func TestRealLoopChecksEmptyQueueReturnsFailedValidation(t *testing.T) {
	checks := &realLoopChecks{}
	if result := checks.Current(context.Background(), domain.Run{}); result.Passed || result.StageID == "" {
		t.Fatalf("empty Current result = %#v", result)
	}
	if result := checks.Final(context.Background(), domain.Run{}); result.Passed || result.StageID == "" {
		t.Fatalf("empty Final result = %#v", result)
	}
}

func (*realLoopChecks) Baseline(context.Context, domain.Run) agent.ValidationResult {
	return agent.ValidationResult{StageID: "unit"}
}
func (c *realLoopChecks) Current(context.Context, domain.Run) agent.ValidationResult {
	if len(c.results) == 0 {
		return agent.ValidationResult{StageID: "missing-validation", Observation: domain.Observation{Stdout: "no validation result configured"}}
	}
	result := c.results[0]
	c.results = c.results[1:]
	return result
}
func (c *realLoopChecks) Final(context.Context, domain.Run) agent.ValidationResult {
	if len(c.results) == 0 {
		return agent.ValidationResult{StageID: "missing-validation", Observation: domain.Observation{Stdout: "no validation result configured"}}
	}
	result := c.results[0]
	c.results = c.results[1:]
	return result
}

type nopController struct{}

func (nopController) Start(context.Context, RunSetup) error                    { return nil }
func (nopController) Approve(context.Context, domain.RunID, string) error      { return nil }
func (nopController) Reject(context.Context, domain.RunID, string, bool) error { return nil }
func (nopController) Cancel(context.Context, domain.RunID) error               { return nil }

type startCountingController struct {
	mu         sync.Mutex
	startCount int
}

func (c *startCountingController) Start(context.Context, RunSetup) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startCount++
	return nil
}
func (*startCountingController) Approve(context.Context, domain.RunID, string) error {
	return nil
}
func (*startCountingController) Reject(context.Context, domain.RunID, string, bool) error {
	return nil
}
func (*startCountingController) Cancel(context.Context, domain.RunID) error { return nil }
func (c *startCountingController) starts() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startCount
}

type cancelRaceController struct {
	store         storeport.Store
	cancelEntered chan struct{}
	allowCancel   chan struct{}
	mu            sync.Mutex
	approveCount  int
}

func newCancelRaceController(st storeport.Store) *cancelRaceController {
	return &cancelRaceController{
		store: st, cancelEntered: make(chan struct{}), allowCancel: make(chan struct{}),
	}
}
func (*cancelRaceController) Start(context.Context, RunSetup) error { return nil }
func (c *cancelRaceController) Approve(context.Context, domain.RunID, string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.approveCount++
	return nil
}
func (*cancelRaceController) Reject(context.Context, domain.RunID, string, bool) error {
	return nil
}
func (c *cancelRaceController) Cancel(ctx context.Context, id domain.RunID) error {
	close(c.cancelEntered)
	select {
	case <-c.allowCancel:
	case <-ctx.Done():
		return ctx.Err()
	}
	return setState(ctx, c.store, id, domain.StateStopped)
}
func (c *cancelRaceController) approves() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.approveCount
}

type blockingController struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingController() *blockingController {
	return &blockingController{started: make(chan struct{}), release: make(chan struct{})}
}

func (c *blockingController) Start(ctx context.Context, _ RunSetup) error {
	c.once.Do(func() { close(c.started) })
	select {
	case <-c.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (*blockingController) Approve(context.Context, domain.RunID, string) error      { return nil }
func (*blockingController) Reject(context.Context, domain.RunID, string, bool) error { return nil }
func (*blockingController) Cancel(context.Context, domain.RunID) error               { return nil }

type terminalController struct {
	store storeport.Store
	mu    sync.Mutex
	last  string
}

func (*terminalController) Start(context.Context, RunSetup) error { return nil }
func (c *terminalController) Approve(ctx context.Context, id domain.RunID, _ string) error {
	c.record("approve")
	return setState(ctx, c.store, id, domain.StateSucceeded)
}
func (c *terminalController) Reject(ctx context.Context, id domain.RunID, _ string, terminate bool) error {
	c.record("reject")
	if terminate {
		return setState(ctx, c.store, id, domain.StateStopped)
	}
	return nil
}
func (c *terminalController) Cancel(ctx context.Context, id domain.RunID) error {
	c.record("cancel")
	return setState(ctx, c.store, id, domain.StateStopped)
}
func (c *terminalController) record(operation string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = operation
}
func (c *terminalController) lastOperation() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

type stateController struct {
	store storeport.Store
	mu    sync.Mutex
	state domain.RunState
}

func (c *stateController) Start(ctx context.Context, setup RunSetup) error {
	c.mu.Lock()
	state := c.state
	c.mu.Unlock()
	return setState(ctx, c.store, setup.Run.ID, state)
}
func (c *stateController) Approve(ctx context.Context, id domain.RunID, _ string) error {
	c.mu.Lock()
	state := c.state
	c.mu.Unlock()
	return setState(ctx, c.store, id, state)
}
func (*stateController) Reject(context.Context, domain.RunID, string, bool) error { return nil }
func (*stateController) Cancel(context.Context, domain.RunID) error               { return nil }

func setState(ctx context.Context, st storeport.Store, id domain.RunID, state domain.RunState) error {
	run, err := st.GetRun(ctx, id)
	if err != nil {
		return err
	}
	run.State = state
	run.UpdatedAt = time.Now().UTC()
	_, err = st.UpdateRun(ctx, run, "TestStateChanged", json.RawMessage(`{}`))
	return err
}
