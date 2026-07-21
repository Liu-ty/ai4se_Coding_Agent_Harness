# AI4SE Coding Agent Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a language-agnostic, validation-driven coding agent harness that applies governed patches, converts objective check failures into structured feedback, and stops only after complete required validation or an explicit terminal condition.

**Architecture:** A Go domain core owns the run state machine, policy, feedback, budgets, and provider-neutral loop. Concrete providers, tools, executors, stores, credentials, HTTP API, React UI, and mock-only public demo connect through explicit ports; local and public profiles use separate composition roots.

**Tech Stack:** Go 1.26.5, React 19.2, TypeScript, Vite 8.1, Node 24 LTS, SQLite via `modernc.org/sqlite`, `github.com/zalando/go-keyring`, `golang.org/x/crypto`, TOML, SSE, GitHub Actions, Docker, Caddy.

## Global Constraints

- Module path is exactly `github.com/Liu-ty/ai4se_Coding_Agent_Harness`; binary name is `ai4se-harness`.
- Go version floor is 1.26.5; frontend build uses Node 24 LTS, React 19.2, and Vite 8.1.
- Release targets are `windows/amd64` and `linux/amd64`.
- Do not use LangChain AgentExecutor, AutoGen, CrewAI, LlamaIndex agents, an SDK agent runner, or any other high-level agent loop.
- All core behavior must be testable with mock/stub providers without network access.
- All implementation follows strict red-green-refactor; no production behavior is added before its failing test is observed.
- Raw shell strings, arbitrary network tools, dependency installation, Git writes, repository escape, binary modification, and credential access are outside the course-delivery action registry.
- The public demo must not register real filesystem, process, provider credential, custom endpoint, or repository-upload capabilities.
- API keys never enter source, Git, SQLite, logs, events, child environments, project configuration, or public-demo state.
- Every task ends with spec-compliance review followed by code-quality review; critical findings are fixed before the task commit.
- Every completed task updates its checkbox, appends its commit hash to the task heading, and records the subagent/human changes in `AGENT_LOG.md`.
- `REFLECTION.md` is authored by the student only. Agents may not draft it; any later AI copy-editing must be disclosed by the student.

---

## Pre-Implementation Cold-Start Gate

This gate occurs after this plan is approved and before Task 1 implementation.

- [ ] Start a new session with an agent product different from the primary development agent.
- [ ] Provide only committed `SPEC.md` and `PLAN.md`; do not provide this conversation, memory, or oral clarification.
- [ ] Ask the cold agent to attempt Task 1 and one of Tasks 2–4, stopping at ambiguity rather than guessing.
- [ ] Record every question, divergent interpretation, failed assumption, and produced artifact in `SPEC_PROCESS.md`.
- [ ] Amend `SPEC.md`/`PLAN.md` with exact before/after diffs for every confirmed defect.
- [ ] Commit the cold-start evidence and revisions before opening an implementation worktree.

Expected gate result: the cold agent can identify every file, interface, command, expected failure, and acceptance criterion needed for the selected tasks without conversation history.

## Worktree and PR Map

| PR/worktree | Tasks | Dependency | May run in parallel with |
|---|---|---|---|
| `foundation` | 1 | cold-start gate | none |
| `config-store-budget` | 2–4 | Task 1 | `policy-tools`, `executor-feedback` |
| `policy-tools` | 5–7 | Task 1 | `config-store-budget`, `executor-feedback` |
| `executor-feedback` | 8–10 | Tasks 1–2 | `config-store-budget`, `policy-tools` |
| `agent-providers` | 11–12 | Tasks 2–10 merged | `credentials-app` after Task 11 interface freezes |
| `credentials-app` | 13–14 | Tasks 2–11 merged | Task 12 |
| `api-ui` | 15–16 | Tasks 11–14 merged | none |
| `demo-release` | 17–18 | Tasks 15–16 merged | none |

Each numbered task is executed by a fresh subagent even when two sequential tasks share one worktree. Review after every task; open one PR per mapped worktree.

## Planned File Structure

```text
cmd/ai4se-harness/             CLI and local/demo composition roots
internal/domain/               Stable entities, enums, state transitions, port types
internal/config/               Strict versioned TOML and platform command resolution
internal/store/                Memory and SQLite run/event/artifact stores
internal/budget/               Decision/mutation/time budgets and progress detection
internal/policy/               Risk classification, profiles, approval digests
internal/workspace/            Canonical paths, protected files, Git baseline
internal/tools/                Registry plus list/search/read/patch/create/check tools
internal/executor/             Cross-platform process execution and mock executor
internal/validation/           Ordered validation pipeline
internal/feedback/             Normalize, classify, redact, fingerprint, compress
internal/provider/             Provider interface, mock, OpenAI-compatible, Anthropic
internal/agent/                Context assembler and agent loop
internal/credentials/          Keyring and encrypted vault
internal/app/                  Run use cases, preflight, repository locking
internal/httpapi/              REST/SSE routes and local web security
internal/demo/                 Fixed mock scenarios and in-memory workspace
web/                           React/Vite application
deploy/                        Docker, Compose, and Caddy assets
scripts/                       Cross-platform test/build entrypoints
.github/workflows/             CI and release workflows
```

## Common Task Exit Gate

Before Task 16, after the task-specific green step every implementer must run:

```powershell
go test ./...
git diff --check
```

From Task 16 onward, build the embedded assets before Go compilation/tests, then run:

```powershell
npm --prefix web run build
npm --prefix web test -- --run
go test ./...
git diff --check
```

Then perform two separate reviews:

1. Compare the diff against the task’s `Interfaces`, `SPEC.md`, and acceptance tests; fix every spec mismatch.
2. Review error paths, names, duplication, platform assumptions, secret handling, and test quality; fix every critical issue.

Only then update `AGENT_LOG.md`, stage explicit files, and commit.

---

### Task 1: Project Skeleton and Run State Machine

**Files:**
- Create: `.gitattributes`
- Create: `.gitignore`
- Create: `go.mod`
- Create: `internal/domain/types.go`
- Create: `internal/domain/state.go`
- Test: `internal/domain/state_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: only Go standard library.
- Produces: `domain.RunID`, `domain.RunState`, `domain.PermissionProfile`, `domain.Action`, `domain.AgentDecision`, `domain.Observation`, `domain.StructuredFeedback`, `domain.Run`, and `domain.Transition(RunState, RunState) error`.

- [ ] **Step 1: Create repository metadata and the minimal Go module**

```text
# .gitattributes
* text=auto
*.go text eol=lf
*.md text eol=lf
*.toml text eol=lf
*.yml text eol=lf
*.yaml text eol=lf

# .gitignore
.env
.env.*
!.env.example
*.db
*.db-shm
*.db-wal
.ai4se-harness/
web/node_modules/
web/dist/
internal/httpapi/webdist/
dist/
coverage/
```

```go
module github.com/Liu-ty/ai4se_Coding_Agent_Harness

go 1.26.5
```

- [ ] **Step 2: Write failing transition tests**

```go
package domain_test

import (
    "testing"
    "github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func TestRepairFlowAllowsBaselineToDecision(t *testing.T) {
    if err := domain.Transition(domain.StateBaselineValidating, domain.StateDeciding); err != nil {
        t.Fatalf("expected valid transition: %v", err)
    }
}

func TestDecisionCannotClaimSuccess(t *testing.T) {
    if err := domain.Transition(domain.StateDeciding, domain.StateSucceeded); err == nil {
        t.Fatal("expected direct success to be rejected")
    }
}

func TestReviewFlowEndsWithoutValidation(t *testing.T) {
    if err := domain.Transition(domain.StateDeciding, domain.StateReviewComplete); err != nil {
        t.Fatalf("expected review completion: %v", err)
    }
}
```

- [ ] **Step 3: Run the tests and observe red**

Run: `go test ./internal/domain -run 'TestRepairFlow|TestDecision|TestReview' -v`  
Expected: FAIL because `internal/domain` and the referenced states do not exist.

- [ ] **Step 4: Implement the stable domain vocabulary and transition table**

```go
package domain

import (
    "encoding/json"
    "errors"
    "time"
)

type RunID string
type RunState string

const (
    StateCreated RunState = "CREATED"
    StatePreflight RunState = "PREFLIGHT"
    StateBaselineValidating RunState = "BASELINE_VALIDATING"
    StateDeciding RunState = "DECIDING"
    StateAwaitingApproval RunState = "AWAITING_APPROVAL"
    StateExecuting RunState = "EXECUTING"
    StateValidating RunState = "VALIDATING"
    StateFinalValidating RunState = "FINAL_VALIDATING"
    StateSucceeded RunState = "SUCCEEDED"
    StateReviewComplete RunState = "REVIEW_COMPLETE"
    StateStopped RunState = "STOPPED"
)

type PermissionProfile string
const (
    ProfileReview PermissionProfile = "review"
    ProfileSupervised PermissionProfile = "supervised"
    ProfileWorkspaceAuto PermissionProfile = "workspace-auto"
)

type Action struct { Kind string `json:"kind"`; Args json.RawMessage `json:"args"` }
type AgentDecision struct { Version string `json:"version"`; Action Action `json:"action"`; ExpectedOutcome string `json:"expected_outcome"` }
type Observation struct { Code string; ExitCode *int; Stdout, Stderr string; StartedAt, EndedAt time.Time; Data json.RawMessage }
type Evidence struct { Source, Message, Path string; Line int }
type StructuredFeedback struct { Category, StageID, Summary, Fingerprint string; Evidence []Evidence; Retryable, OutputTruncated bool; PreviousOccurrences int }
type Run struct { ID RunID; State RunState; Profile PermissionProfile; Task, RepoRoot, CurrentStage string; CreatedAt, UpdatedAt time.Time }

var ErrInvalidTransition = errors.New("invalid run state transition")
```

```go
package domain

func Transition(from, to RunState) error {
    allowed := map[RunState]map[RunState]bool{
        StateCreated: {StatePreflight: true},
        StatePreflight: {StateBaselineValidating: true, StateDeciding: true, StateStopped: true},
        StateBaselineValidating: {StateDeciding: true, StateStopped: true},
        StateDeciding: {StateAwaitingApproval: true, StateExecuting: true, StateReviewComplete: true, StateStopped: true},
        StateAwaitingApproval: {StateExecuting: true, StateDeciding: true, StateStopped: true},
        StateExecuting: {StateDeciding: true, StateValidating: true, StateStopped: true},
        StateValidating: {StateDeciding: true, StateFinalValidating: true, StateStopped: true},
        StateFinalValidating: {StateSucceeded: true, StateDeciding: true, StateStopped: true},
    }
    if !allowed[from][to] { return ErrInvalidTransition }
    return nil
}
```

- [ ] **Step 5: Run green and commit**

Run: `go test ./internal/domain -v`  
Expected: PASS.

Commit:

```powershell
git add .gitattributes .gitignore go.mod internal/domain AGENT_LOG.md
git commit -m "feat: define harness domain state machine"
```

---

### Task 2: Strict Versioned Project Configuration

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/load.go`
- Create: `internal/config/resolve.go`
- Test: `internal/config/config_test.go`
- Create: `testdata/config/valid.toml`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `domain.PermissionProfile`.
- Produces: `config.Config`, `config.CommandSpec`, `config.Load(io.Reader) (Config, error)`, and `config.ResolveStage(ValidationStage, runtime.GOOS) (CommandSpec, error)`.

- [ ] **Step 1: Write failing strict-load and platform-resolution tests**

```go
package config_test

func TestLoadRejectsUnknownField(t *testing.T) {
    _, err := config.Load(strings.NewReader("version = 1\nunknown = true\n"))
    if !errors.Is(err, config.ErrUnknownField) { t.Fatalf("got %v", err) }
}

func TestResolveWindowsOverride(t *testing.T) {
    stage := config.ValidationStage{ID:"unit-test", Executable:"go", Args:[]string{"test","./..."}, Windows:&config.CommandOverride{Executable:"go.exe"}}
    got, err := config.ResolveStage(stage, "windows")
    if err != nil || got.Executable != "go.exe" { t.Fatalf("got %#v, %v", got, err) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/config -v`  
Expected: FAIL because package `internal/config` does not exist.

- [ ] **Step 3: Implement exact configuration types**

```go
type Config struct {
    Version int `toml:"version"`
    DefaultProfile domain.PermissionProfile `toml:"default_profile"`
    Budget BudgetConfig `toml:"budget"`
    Validation []ValidationStage `toml:"validation"`
    Policy PolicyConfig `toml:"policy"`
}
type BudgetConfig struct { MaxDecisions, MaxMutations, MaxProtocolRepairs int; WallClock string }
type CommandOverride struct { Executable string; Args []string }
type ValidationStage struct {
    ID, Kind, Executable, WorkingDirectory string
    Args []string
    Timeout string
    MaxOutputBytes int
    Required bool
    Windows, Linux *CommandOverride
    Classifiers []ClassifierRule
}
type ClassifierRule struct { Category, Pattern string }
type PolicyConfig struct { MaxFiles, MaxChangedLines, MaxFileBytes int; Protected []string }
type CommandSpec struct { ID, Kind, Executable, WorkingDirectory string; Args []string; Timeout time.Duration; MaxOutputBytes int; Required bool }
```

Use `github.com/BurntSushi/toml` metadata to reject undecoded keys, require `version = 1`, reject duplicate stage IDs, reject absolute working directories, reject empty executables, and parse positive timeouts.

- [ ] **Step 4: Add the canonical fixture and run green**

```toml
version = 1
default_profile = "workspace-auto"

[budget]
max_decisions = 30
max_mutations = 5
max_protocol_repairs = 2
wall_clock = "20m"

[[validation]]
id = "unit-test"
kind = "targeted-test"
executable = "go"
args = ["test", "./..."]
working_directory = "."
timeout = "2m"
max_output_bytes = 262144
required = true

[validation.windows]
executable = "go.exe"
```

Run: `go test ./internal/config -v`  
Expected: PASS, including strict unknown-field and platform override cases.

- [ ] **Step 5: Commit**

```powershell
git add internal/config testdata/config AGENT_LOG.md go.mod go.sum
git commit -m "feat: add strict versioned harness configuration"
```

---

### Task 3: Hash-Chained Memory and SQLite Event Store

**Files:**
- Create: `internal/domain/events.go`
- Create: `internal/store/store.go`
- Create: `internal/store/memory.go`
- Create: `internal/store/sqlite.go`
- Create: `internal/store/migrations/001_init.sql`
- Test: `internal/store/store_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `domain.Run`, `domain.RunEvent`, `domain.Artifact`.
- Produces: `store.Store` with `CreateRun`, `AppendEvent`, `GetRun`, `ListEvents`, and `PutArtifact`; memory and SQLite implementations must pass one contract suite.

- [ ] **Step 1: Write a store contract that fails for both implementations**

```go
type factory func(t *testing.T) store.Store
func contract(t *testing.T, newStore factory) {
    s := newStore(t)
    run := domain.Run{ID:"run-1", State:domain.StateCreated}
    if err := s.CreateRun(context.Background(), run); err != nil { t.Fatal(err) }
    e1, err := s.AppendEvent(context.Background(), run.ID, "RunCreated", json.RawMessage(`{"ok":true}`))
    if err != nil { t.Fatal(err) }
    e2, err := s.AppendEvent(context.Background(), run.ID, "StateChanged", json.RawMessage(`{"state":"PREFLIGHT"}`))
    if err != nil { t.Fatal(err) }
    if e2.PreviousHash != e1.Hash || e2.Sequence != 2 { t.Fatalf("broken chain: %#v %#v", e1, e2) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/store -v`  
Expected: FAIL because store/domain event types are missing.

- [ ] **Step 3: Define events and store contract**

```go
type RunEvent struct { RunID RunID; Sequence uint64; Type string; At time.Time; Payload json.RawMessage; PreviousHash, Hash string }
type Artifact struct { ID, RunID, Kind, SHA256 string; Content []byte; Truncated bool }
```

```go
type Store interface {
    CreateRun(context.Context, domain.Run) error
    UpdateRun(context.Context, domain.Run, string, json.RawMessage) (domain.RunEvent, error)
    AppendEvent(context.Context, domain.RunID, string, json.RawMessage) (domain.RunEvent, error)
    GetRun(context.Context, domain.RunID) (domain.Run, error)
    ListEvents(context.Context, domain.RunID, uint64) ([]domain.RunEvent, error)
    PutArtifact(context.Context, domain.Artifact) error
}
```

Hash input is canonical `run_id | sequence | type | unix_nano | payload | previous_hash`, encoded with length prefixes and SHA-256.

- [ ] **Step 4: Implement memory then SQLite transaction semantics**

The SQLite migration creates `runs`, `run_events`, `artifacts`, and `schema_migrations`, unique on `(run_id, sequence)`. `UpdateRun` updates the run snapshot and appends the event inside one SQL transaction. Use `modernc.org/sqlite`; configure one writer, foreign keys, busy timeout, and WAL for local mode.

- [ ] **Step 5: Run contract tests, reopen SQLite, and commit**

Run: `go test ./internal/store -v`  
Expected: PASS for memory, SQLite, concurrent sequence allocation, rollback-on-event-failure, and reopen persistence.

```powershell
git add internal/domain/events.go internal/store AGENT_LOG.md go.mod go.sum
git commit -m "feat: persist hash-chained run events"
```

---

### Task 4: Dual Budgets and No-Progress Detection

**Files:**
- Create: `internal/budget/tracker.go`
- Create: `internal/budget/progress.go`
- Test: `internal/budget/tracker_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: configured limits, clock, feedback fingerprint, diff digest.
- Produces: `budget.Tracker`, `budget.ProgressDetector`, and typed stop reasons.

- [ ] **Step 1: Write failing boundary tests with a fake clock**

```go
func TestTrackerStopsAtDecisionLimit(t *testing.T) {
    tr := budget.New(budget.Limits{MaxDecisions:2, MaxMutations:5, WallClock:20*time.Minute}, fakeClock{})
    if err := tr.RecordDecision(); err != nil { t.Fatal(err) }
    if err := tr.RecordDecision(); err != nil { t.Fatal(err) }
    if !errors.Is(tr.RecordDecision(), budget.ErrDecisionBudget) { t.Fatal("expected decision budget") }
}
func TestNoProgressNeedsSameFailureAndSameDiffTwice(t *testing.T) {
    p := budget.NewProgressDetector(2)
    if p.Observe("fp", "diff-a") { t.Fatal("first observation cannot stop") }
    if !p.Observe("fp", "diff-a") { t.Fatal("second identical observation must stop") }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/budget -v`  
Expected: FAIL because budget package is missing.

- [ ] **Step 3: Implement tracker and detector**

```go
type Limits struct { MaxDecisions, MaxMutations, MaxProtocolRepairs int; WallClock time.Duration }
type Usage struct { Decisions, Mutations, ProtocolRepairs int; StartedAt time.Time }
type Clock interface { Now() time.Time }
type Tracker struct { limits Limits; usage Usage; clock Clock }
func (t *Tracker) RecordDecision() error
func (t *Tracker) RecordMutation() error
func (t *Tracker) RecordProtocolRepair() error
func (t *Tracker) CheckTime() error
func (t *Tracker) Snapshot() Usage
```

`ProgressDetector.Observe` compares the last failure and diff digests and returns true only at the configured consecutive threshold.

- [ ] **Step 4: Run green and commit**

Run: `go test ./internal/budget -v`  
Expected: PASS, including exact-limit, wall-clock, reset-after-progress, and same-failure/different-diff warning cases.

```powershell
git add internal/budget AGENT_LOG.md
git commit -m "feat: enforce run budgets and progress stops"
```

---

### Task 5: Policy Engine, Permission Profiles, and Exact Approvals

**Files:**
- Create: `internal/policy/policy.go`
- Create: `internal/policy/risk.go`
- Create: `internal/policy/approval.go`
- Test: `internal/policy/policy_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `domain.Action`, run/profile/baselines, workspace risk facts.
- Produces: `policy.Engine.Evaluate(Context, Action) Decision`, `policy.ApprovalDigest`, and one-use `ApprovalStore`.

- [ ] **Step 1: Write the failing profile matrix and digest tests**

```go
func TestPatchProfileMatrix(t *testing.T) {
    action := domain.Action{Kind:"apply_patch", Args:json.RawMessage(`{"path":"a.go"}`)}
    cases := []struct{ profile domain.PermissionProfile; want policy.Verdict }{
        {domain.ProfileReview, policy.Deny},
        {domain.ProfileSupervised, policy.RequireApproval},
        {domain.ProfileWorkspaceAuto, policy.Allow},
    }
    for _, tc := range cases {
        got := policy.NewEngine().Evaluate(policy.Context{Profile:tc.profile}, action)
        if got.Verdict != tc.want { t.Fatalf("%s: %s", tc.profile, got.Verdict) }
    }
}
func TestApprovalDigestChangesWithBaseline(t *testing.T) {
    a := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"a.go":"one"})
    b := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"a.go":"two"})
    if a == b { t.Fatal("digest must bind baseline") }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/policy -v`  
Expected: FAIL because policy types are missing.

- [ ] **Step 3: Implement risk-first policy evaluation**

```go
type Verdict string
const ( Allow Verdict = "ALLOW"; RequireApproval Verdict = "REQUIRE_APPROVAL"; Deny Verdict = "DENY" )
type Risk string
const ( RiskNormal Risk="NORMAL"; RiskGuarded Risk="GUARDED"; RiskHardDenied Risk="HARD_DENIED" )
type Context struct { RunID domain.RunID; Profile domain.PermissionProfile; RepoRoot string; Dirty bool; Baselines map[string]string }
type Decision struct { Verdict Verdict; Risk Risk; Code, Message, Digest string }
type Engine struct{}
func (Engine) Evaluate(Context, domain.Action) Decision
```

Hard-deny unknown/raw-shell/network/credential/repository-escape facts before applying profile mappings. Guard large/protected/dirty-worktree mutations. A review-mode patch is denied but its redacted request can later be saved as a proposal artifact.

- [ ] **Step 4: Implement one-use approval consumption**

`ApprovalStore.Grant(digest)` records one digest; `Consume(digest)` succeeds exactly once. A changed action/profile/baseline produces a new digest and cannot consume the old grant.

- [ ] **Step 5: Run green and commit**

Run: `go test ./internal/policy -v`  
Expected: PASS for profile matrix, hard denials, guarded limits, dirty workspace, digest binding, one-use consumption, and repository config unable to loosen policy.

```powershell
git add internal/policy AGENT_LOG.md
git commit -m "feat: add selectable policy profiles and approvals"
```

---

### Task 6: Canonical Workspace and Read-Only Tools

**Files:**
- Create: `internal/workspace/path.go`
- Create: `internal/workspace/git.go`
- Create: `internal/tools/tool.go`
- Create: `internal/tools/registry.go`
- Create: `internal/tools/read.go`
- Create: `internal/tools/search.go`
- Test: `internal/tools/read_test.go`
- Test: `internal/workspace/path_test.go`
- Create: `internal/testutil/testrepo/repo.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: repository root and typed action arguments.
- Produces: `tools.Tool`, `tools.Registry`, `workspace.Resolve`, `workspace.GitBaseline`, list/search/read observations with SHA-256, and a shared temporary-Git-repository test helper.

- [ ] **Step 1: Write path-escape, symlink, binary, and truncation tests**

```go
func TestResolveRejectsParentEscape(t *testing.T) {
    _, err := workspace.Resolve(t.TempDir(), "../secret")
    if !errors.Is(err, workspace.ErrOutsideRoot) { t.Fatalf("got %v", err) }
}
func TestReadReturnsHashAndTruncation(t *testing.T) {
    root := t.TempDir(); os.WriteFile(filepath.Join(root,"a.txt"), []byte("abcdef"), 0600)
    got, err := tools.NewReadTool(root, 3).Execute(context.Background(), json.RawMessage(`{"path":"a.txt"}`))
    if err != nil || !got.Truncated || got.SHA256 == "" || got.Text != "abc" { t.Fatalf("%#v %v", got, err) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/workspace ./internal/tools -v`  
Expected: FAIL because packages are missing.

- [ ] **Step 3: Implement canonical path resolution**

Resolve the root with `filepath.EvalSymlinks`, join/clean the relative path, resolve the nearest existing parent for new paths, and compare with `filepath.Rel`. Reject absolute inputs, `..` escape, `.git`, protected secret globs, and symlinks whose resolved target leaves the root.

- [ ] **Step 4: Implement tool contract and registry**

```go
type Result struct { Code string; Data json.RawMessage; Text, SHA256 string; Truncated bool }
type Tool interface { Kind() string; Execute(context.Context, json.RawMessage) (Result, error) }
type Registry struct { tools map[string]Tool }
func NewRegistry(list ...Tool) (*Registry, error)
func (r *Registry) Execute(ctx context.Context, action domain.Action) (Result, error)
```

Implement list with bounded `filepath.WalkDir`, search with Go regular expressions and bounded matches, and read with UTF-8/binary detection, byte limits, SHA-256, and deterministic ordering.

Implement `testrepo.New(t, files)` under `internal/testutil/testrepo`: create the files, run `git init`, configure a test-only name/email locally, add, and commit. Provide `Root`, `Read`, and `Write` helpers used by later tool/app integration tests.

- [ ] **Step 5: Run green and commit**

Run: `go test ./internal/workspace ./internal/tools -v`  
Expected: PASS for escape, symlink, `.git`, protected files, binary, stable ordering, result limits, invalid regex, hash, and truncation.

```powershell
git add internal/workspace internal/tools AGENT_LOG.md
git commit -m "feat: add bounded repository read tools"
```

---

### Task 7: Safe Patch and File-Creation Tools

**Files:**
- Create: `internal/tools/patch.go`
- Create: `internal/tools/create.go`
- Create: `internal/tools/patch_headers.go`
- Test: `internal/tools/patch_test.go`
- Test: `internal/tools/create_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `workspace.Resolve`, baseline SHA-256 map, policy limits, installed Git executable.
- Produces: `apply_patch` and `create_file` tools returning changed paths and diff digests; no partial mutation on failure.

- [ ] **Step 1: Write failing stale-baseline, conflict, protected-path, atomicity, and no-overwrite tests**

```go
func TestPatchRejectsStaleBaselineWithoutMutation(t *testing.T) {
    repo := testrepo.New(t, map[string]string{"a.txt":"old\n"})
    tool := tools.NewPatchTool(repo.Root, tools.PatchLimits{MaxFiles:5, MaxChangedLines:500})
    args := `{"patch":"--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-old\n+new\n","baselines":{"a.txt":"deadbeef"}}`
    _, err := tool.Execute(context.Background(), json.RawMessage(args))
    if !errors.Is(err, tools.ErrStaleBaseline) { t.Fatalf("got %v", err) }
    if got := repo.Read("a.txt"); got != "old\n" { t.Fatalf("mutated: %q", got) }
}
func TestCreateNeverOverwrites(t *testing.T) {
    root := t.TempDir(); os.WriteFile(filepath.Join(root,"a.txt"), []byte("old"), 0600)
    _, err := tools.NewCreateTool(root, 1024).Execute(context.Background(), json.RawMessage(`{"path":"a.txt","content":"new"}`))
    if !errors.Is(err, tools.ErrAlreadyExists) { t.Fatalf("got %v", err) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/tools -run 'TestPatch|TestCreate' -v`  
Expected: FAIL because patch/create tools do not exist.

- [ ] **Step 3: Implement deterministic patch-header parsing and validation**

Parse only unified-diff file headers and hunks. Reject rename, delete, binary patch, absolute paths, `/dev/null` for existing-file patching, duplicate paths, more than five files, and more than 500 added/deleted lines. Resolve every path through `workspace.Resolve` and compare every supplied baseline hash before invoking Git.

- [ ] **Step 4: Implement all-or-nothing Git apply and create**

Run `git apply --check --whitespace=nowarn -` with patch bytes on stdin; only on success run `git apply --whitespace=nowarn -` without `--reject`. Recheck all baselines immediately before the real apply. Git’s non-reject path must leave all targets unchanged on failure; verify this against captured hashes and escalate any unexpected mutation as `PATCH_ATOMICITY_BREACH` without running reset/clean. Return a sorted path list plus SHA-256 of the resulting `git diff --binary` output.

`create_file` opens the target directly with `O_CREATE|O_EXCL`, writes, fsyncs, and closes after policy and size checks. On write/sync failure it removes the newly created path; it never overwrites an existing path.

- [ ] **Step 5: Run green and commit**

Run: `go test ./internal/tools -run 'TestPatch|TestCreate' -v`  
Expected: PASS for clean apply, conflict, stale baseline, atomic injected failure, delete/rename/binary rejection, protected path, limits, and no overwrite.

```powershell
git add internal/tools AGENT_LOG.md
git commit -m "feat: apply bounded atomic code patches"
```

---

### Task 8: Cross-Platform Restricted Process Executor

**Files:**
- Create: `internal/executor/executor.go`
- Create: `internal/executor/local.go`
- Create: `internal/executor/process_linux.go`
- Create: `internal/executor/process_windows.go`
- Create: `internal/executor/mock.go`
- Test: `internal/executor/executor_test.go`
- Create: `internal/executor/testhelper/main.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `config.CommandSpec`, context cancellation, sanitized environment.
- Produces: `executor.Executor.Run(context.Context, config.CommandSpec) (domain.Observation, error)` with bounded separate stdout/stderr and process-tree cleanup.

- [ ] **Step 1: Write failing exit, truncation, timeout, environment, and process-tree tests**

```go
func TestExecutorCapturesSeparateStreams(t *testing.T) {
    spec := helperSpec(t, "streams")
    got, err := executor.NewLocal().Run(context.Background(), spec)
    if err != nil || got.Stdout != "out\n" || got.Stderr != "err\n" || *got.ExitCode != 7 { t.Fatalf("%#v %v", got, err) }
}
func TestExecutorDoesNotForwardSecretEnvironment(t *testing.T) {
    t.Setenv("OPENAI_API_KEY", "canary-secret")
    got, _ := executor.NewLocal().Run(context.Background(), helperSpec(t, "env"))
    if strings.Contains(got.Stdout, "canary-secret") { t.Fatal("secret inherited") }
}
```

- [ ] **Step 2: Run red on the current platform**

Run: `go test ./internal/executor -v`  
Expected: FAIL because executor is missing.

- [ ] **Step 3: Implement bounded process execution**

```go
type Executor interface { Run(context.Context, config.CommandSpec) (domain.Observation, error) }
type Local struct { BaseEnv []string; Clock func() time.Time }
type Mock struct { Results map[string][]domain.Observation; Calls []config.CommandSpec }
```

Use `exec.CommandContext(spec.Executable, spec.Args...)`, set a canonical repository-relative working directory resolved earlier, retain an explicit safe environment set, remove names matching `KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL`, and cap stdout/stderr independently while continuing to drain pipes.

- [ ] **Step 4: Implement OS process-tree controllers**

Linux build-tag file sets `SysProcAttr.Setpgid = true`; cancellation sends SIGTERM to `-pid`, waits 2 seconds, then SIGKILL. Windows build-tag file creates a Job Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, assigns the child process, and closes the job on cancellation. Use `golang.org/x/sys/windows`; do not invoke `taskkill`, PowerShell, or `cmd`.

- [ ] **Step 5: Run green on Windows and Linux CI-compatible tests and commit**

Run: `go test ./internal/executor -v`  
Expected: PASS for streams, exit code, output limit, timeout, cancellation, secret environment, and child-process cleanup on the current OS.

```powershell
git add internal/executor AGENT_LOG.md go.mod go.sum
git commit -m "feat: execute bounded checks across platforms"
```

---

### Task 9: Ordered Validation Pipeline

**Files:**
- Create: `internal/validation/pipeline.go`
- Create: `internal/validation/result.go`
- Test: `internal/validation/pipeline_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: resolved `[]config.CommandSpec` and `executor.Executor`.
- Produces: `validation.Pipeline.RunStage`, `RunFrom`, and `RunAllRequired`, with stable stage results.

- [ ] **Step 1: Write failing fail-fast and final-rerun tests**

```go
func TestRunFromStopsAtFirstFailure(t *testing.T) {
    ex := &executor.Mock{Results: map[string][]domain.Observation{
        "targeted": {{Code:"EXIT", ExitCode:intp(0)}},
        "full": {{Code:"EXIT", ExitCode:intp(1)}},
        "lint": {{Code:"EXIT", ExitCode:intp(0)}},
    }}
    p := validation.New(stages("targeted","full","lint"), ex)
    got := p.RunFrom(context.Background(), 0)
    if got.FailedStage != "full" || len(ex.Calls) != 2 { t.Fatalf("%#v %#v", got, ex.Calls) }
}
func TestFinalValidationRerunsAllRequired(t *testing.T) {
    ex := &executor.Mock{Results: map[string][]domain.Observation{
        "targeted": {{Code:"EXIT", ExitCode:intp(0)}},
        "full": {{Code:"EXIT", ExitCode:intp(0)}},
        "lint": {{Code:"EXIT", ExitCode:intp(0)}},
    }}
    p := validation.New(stages("targeted","full","lint"), ex)
    got := p.RunAllRequired(context.Background())
    if !got.Complete || callIDs(ex.Calls) != "targeted,full,lint" { t.Fatalf("%#v %#v", got, ex.Calls) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/validation -v`  
Expected: FAIL because validation package is missing.

- [ ] **Step 3: Implement pipeline result types and ordered execution**

```go
type StageResult struct { Stage config.CommandSpec; Observation domain.Observation; Passed bool }
type Result struct { Stages []StageResult; FailedStage string; Complete bool }
type Pipeline struct { stages []config.CommandSpec; executor executor.Executor }
func (p *Pipeline) RunStage(context.Context, int) StageResult
func (p *Pipeline) RunFrom(context.Context, int) Result
func (p *Pipeline) RunAllRequired(context.Context) Result
```

Pass means process started, did not time out/cancel, and exit code is zero. Optional-stage failure is recorded but does not prevent required stages from running; any required failure stops the current pass.

- [ ] **Step 4: Run green and commit**

Run: `go test ./internal/validation -v`  
Expected: PASS for ordered stages, first required failure, optional failure, context cancellation, missing exit, and complete final rerun.

```powershell
git add internal/validation AGENT_LOG.md
git commit -m "feat: add staged validation pipeline"
```

---

### Task 10: Deterministic Feedback Pipeline

**Files:**
- Create: `internal/feedback/normalize.go`
- Create: `internal/feedback/classify.go`
- Create: `internal/feedback/fingerprint.go`
- Create: `internal/feedback/redact.go`
- Create: `internal/feedback/compress.go`
- Test: `internal/feedback/feedback_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: action/tool/validation observations and configured classifier rules.
- Produces: `feedback.Pipeline.Process(Input) domain.StructuredFeedback`, stable secret-safe fingerprints, and bounded evidence.

- [ ] **Step 1: Write failing normalization, redaction, category, and fingerprint tests**

```go
func TestFingerprintIgnoresANSIPathsTimingAndAddresses(t *testing.T) {
    a := "\x1b[31mFAIL pkg/a_test.go:12 took 1.24s ptr=0xabc123\x1b[0m"
    b := "FAIL pkg/a_test.go:99 took 9.87s ptr=0xdef456"
    if feedback.Fingerprint("unit-test", "TEST_FAILURE", a) != feedback.Fingerprint("unit-test", "TEST_FAILURE", b) { t.Fatal("unstable fingerprint") }
}
func TestRedactorRemovesKnownAndPatternSecrets(t *testing.T) {
    r := feedback.NewRedactor([]string{"exact-canary"})
    got := r.Redact("Authorization: Bearer exact-canary OPENAI_API_KEY=sk-live-123456789")
    if strings.Contains(got,"exact-canary") || strings.Contains(got,"sk-live") { t.Fatalf("leaked: %s", got) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/feedback -v`  
Expected: FAIL because feedback package is missing.

- [ ] **Step 3: Implement pure pipeline stages**

Classification priority is protocol → policy → patch → environment → configured validation regex → generic validation → progress. Strip ANSI; normalize slashes; replace line numbers, durations, hex addresses, UUIDs, and temporary roots before hashing. Redact exact runtime secrets before regex patterns. Preserve only the first and last bounded evidence groups and mark truncation.

```go
type Input struct { StageID, Code string; Observation domain.Observation; Rules []config.ClassifierRule; PriorOccurrences int; Secrets []string }
type Pipeline struct { MaxEvidence int; MaxSummaryBytes int }
func (p Pipeline) Process(Input) domain.StructuredFeedback
```

- [ ] **Step 4: Add table tests for all required categories**

Cases must cover invalid JSON, unknown action, approval required, path denial, stale patch, missing executable, timeout, cancellation, test/compile/type/lint/build, generic exit, empty patch, regression, and output truncation.

- [ ] **Step 5: Run green and commit**

Run: `go test ./internal/feedback -v`  
Expected: PASS with no canary secret in failure output (`go test` output included).

```powershell
git add internal/feedback AGENT_LOG.md
git commit -m "feat: classify and compress validation feedback"
```

---

### Task 11: Provider-Neutral Agent Loop and Conditional Mock E2E

**Files:**
- Create: `internal/provider/provider.go`
- Create: `internal/provider/mock.go`
- Create: `internal/agent/context.go`
- Create: `internal/agent/loop.go`
- Test: `internal/agent/loop_test.go`
- Create: `internal/agent/testdata/failing_repo/bug.go`
- Create: `internal/agent/testdata/failing_repo/bug_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: store, provider, registry, policy, approvals, validation, feedback, budget, and context assembler.
- Produces: `provider.Provider.Decide`, `agent.Loop.Run`, `agent.Loop.ResumeApproval`, and the deterministic two-patch mechanism test.

- [ ] **Step 1: Write the failing end-to-end loop test**

```go
func TestLoopUsesFailureToChangeNextPatch(t *testing.T) {
    env := newHarnessFixture(t)
    result, err := env.Loop.Run(context.Background(), env.Run)
    if err != nil { t.Fatal(err) }
    if result.State != domain.StateSucceeded { t.Fatalf("%#v", result) }
    calls := env.Provider.Calls()
    if len(calls) < 3 || calls[2].Request.LastFeedback.Fingerprint == "" { t.Fatal("corrective decision did not receive test feedback") }
    if sha256.Sum256(calls[1].Returned.Decision.Action.Args) == sha256.Sum256(calls[2].Returned.Decision.Action.Args) { t.Fatal("patch action did not change") }
    assertEventTypes(t, env.Store, "PolicyDenied", "ValidationFailed", "FeedbackProduced", "ValidationPassed", "RunSucceeded")
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/agent -run TestLoopUsesFailure -v`  
Expected: FAIL because provider and loop packages are missing.

- [ ] **Step 3: Define provider request/response and conditional mock**

```go
type Request struct { Task string; Context []ContextItem; LastFeedback *domain.StructuredFeedback; AllowedActions []string }
type Response struct { Decision domain.AgentDecision; Usage Usage }
type Provider interface { Decide(context.Context, Request) (Response, error) }
type Usage struct { InputTokens, OutputTokens int }
type ContextItem struct { Kind, Label, Content, SHA256 string }
type MockCall struct { Request Request; Returned Response }
```

The mock returns a raw-shell action when no feedback exists; on `POLICY_DENIED` it returns an intentionally incomplete patch; only when `Request.LastFeedback.Category == "TEST_FAILURE"` and evidence names the expected failure does it return the corrected patch, followed by `finish` after passing validation. `Mock.Calls() []MockCall` records both request and response. This makes behavior conditional, not a pre-scripted sequence.

- [ ] **Step 4: Implement one-action loop orchestration**

For repair runs: preflight state already exists → baseline failure → decide → parse/record → policy → approval/execute → observation → feedback → automatic current-stage validation after mutation → budget/progress → decide; `finish` invokes full validation. Review profile skips baseline/checks, executes reads/searches only, records a denied patch as proposal artifact, and ends `REVIEW_COMPLETE`.

Persist a typed event at every boundary. Never trust `finish` to set success directly. Use the transition table for every state change.

- [ ] **Step 5: Add protocol-repair, approval, no-progress, cancellation, and final-regression tests**

Each test uses mocks and asserts exact terminal state plus event order. The protocol test returns invalid JSON twice and then asserts `PROTOCOL_EXHAUSTED`; the final-regression test passes the active stage but fails the full pipeline and returns to deciding.

- [ ] **Step 6: Run green and commit**

Run: `go test ./internal/agent -v`  
Expected: PASS; no network and no real LLM.

```powershell
git add internal/provider internal/agent AGENT_LOG.md
git commit -m "feat: close the mock-driven repair loop"
```

---

### Task 12: OpenAI-Compatible and Anthropic Provider Adapters

**Files:**
- Create: `internal/provider/httpclient.go`
- Create: `internal/provider/openai.go`
- Create: `internal/provider/anthropic.go`
- Test: `internal/provider/openai_test.go`
- Test: `internal/provider/anthropic_test.go`
- Test: `internal/provider/contract_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `provider.Request`, endpoint-bound credential supplier, `http.Client`.
- Produces: both adapters satisfying `provider.Provider`; errors normalize to auth, rate-limit, invalid-response, timeout, and transport codes.

- [ ] **Step 1: Write local HTTP contract tests before adapters**

```go
func providerContract(t *testing.T, build func(url string, client *http.Client) provider.Provider, handler http.Handler) {
    srv := httptest.NewServer(handler); defer srv.Close()
    p := build(srv.URL, srv.Client())
    got, err := p.Decide(context.Background(), canonicalRequest())
    if err != nil { t.Fatal(err) }
    if got.Decision.Version != "1" || got.Decision.Action.Kind != "read_file" { t.Fatalf("%#v", got) }
}
```

Add provider-specific handlers that assert OpenAI-compatible `/v1/chat/completions` bearer auth and Anthropic `/v1/messages` `x-api-key` plus version header. Return canonical decision JSON as model text.

- [ ] **Step 2: Run red**

Run: `go test ./internal/provider -run 'OpenAI|Anthropic|Contract' -v`  
Expected: FAIL because adapters are missing.

- [ ] **Step 3: Implement shared safe HTTP behavior**

Use injected `http.Client`, context deadlines, a 1 MiB response limit, no automatic cross-host redirect with Authorization, normalized endpoint host, JSON content type validation, and bounded error bodies. Credential supplier signature:

```go
type CredentialSource interface { Get(context.Context, string, string) ([]byte, error) }
```

Zero the returned byte slice after constructing the request. Never include headers or request bodies in returned errors.

- [ ] **Step 4: Implement both request/response mappings**

OpenAI-compatible sends a system message containing the canonical JSON schema and user/context messages; Anthropic sends the same protocol through top-level system plus messages. Parse only textual JSON into `domain.AgentDecision`, validate protocol version and exactly one action, and return usage fields.

- [ ] **Step 5: Test error normalization and commit**

Tests: 401, 429 with retry metadata, 500 bounded body, timeout, redirect-to-other-host, malformed JSON, missing content, oversized response, and key never present in error output.

Run: `go test ./internal/provider -v`  
Expected: PASS without external network.

```powershell
git add internal/provider AGENT_LOG.md
git commit -m "feat: connect OpenAI-compatible and Anthropic providers"
```

---

### Task 13: OS Keyring and Encrypted Vault Credentials

**Files:**
- Create: `internal/credentials/store.go`
- Create: `internal/credentials/keyring.go`
- Create: `internal/credentials/vault.go`
- Create: `internal/credentials/service.go`
- Test: `internal/credentials/vault_test.go`
- Test: `internal/credentials/service_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: provider ID, normalized endpoint host, hidden key bytes, optional master-password callback.
- Produces: `credentials.Store`, `credentials.Service.Add/Status/Update/Clear/Get`, keyring-first selection, and Argon2id/XChaCha20-Poly1305 fallback.

- [ ] **Step 1: Write failing vault and status tests with canary secrets**

```go
func TestVaultRoundTripAndWrongPassword(t *testing.T) {
    path := filepath.Join(t.TempDir(), "vault.bin")
    v := credentials.NewVault(path, func() ([]byte,error) { return []byte("master-pass"), nil })
    if err := v.Set(context.Background(), credentials.Ref{Provider:"openai", Host:"api.openai.com"}, []byte("canary-key")); err != nil { t.Fatal(err) }
    got, err := v.Get(context.Background(), credentials.Ref{Provider:"openai", Host:"api.openai.com"})
    if err != nil || string(got) != "canary-key" { t.Fatalf("%q %v", got, err) }
    bad := credentials.NewVault(path, func() ([]byte,error) { return []byte("wrong"), nil })
    if _, err := bad.Get(context.Background(), credentials.Ref{Provider:"openai", Host:"api.openai.com"}); !errors.Is(err, credentials.ErrDecrypt) { t.Fatalf("%v", err) }
}
func TestStatusNeverContainsSecret(t *testing.T) {
    svc := credentials.NewService(credentials.NewMemoryStore(), nil)
    ref := credentials.Ref{Provider:"openai", Host:"api.openai.com"}
    if err := svc.Add(context.Background(), ref, []byte("canary-key")); err != nil { t.Fatal(err) }
    status, err := svc.Status(context.Background(), ref)
    if err != nil { t.Fatal(err) }
    raw, _ := json.Marshal(status)
    if bytes.Contains(raw, []byte("canary-key")) { t.Fatalf("leaked: %s", raw) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/credentials -v`  
Expected: FAIL because credential package is missing.

- [ ] **Step 3: Implement store and service contracts**

```go
type Ref struct { Provider, Host string }
type Status struct { Ref Ref `json:"ref"`; Configured bool `json:"configured"`; Backend string `json:"backend"`; UpdatedAt time.Time `json:"updated_at"` }
type Store interface { Set(context.Context, Ref, []byte) error; Get(context.Context, Ref) ([]byte,error); Delete(context.Context, Ref) error; Status(context.Context, Ref) (Status,error) }
type CredentialService interface {
    Add(context.Context, Ref, []byte) error
    Update(context.Context, Ref, []byte) error
    Status(context.Context, Ref) (Status, error)
    Clear(context.Context, Ref) error
    Get(context.Context, string, string) ([]byte, error)
}
type Service struct { primary Store; fallback Store }
```

Use `github.com/zalando/go-keyring` for the primary adapter. Treat Secret Service unavailable/locked errors as fallback-eligible, not invalid-key errors. Provider and host form the keyring account identity.

- [ ] **Step 4: Implement the exact vault format**

Binary format: magic `A4SEVLT1`, version byte, 16-byte random salt, Argon2id parameters, 24-byte XChaCha nonce, ciphertext. Use Argon2id with time=3, memory=64 MiB, threads=2, key length=32; use XChaCha20-Poly1305 and authenticate provider/host as associated data. Write through an owner-only temporary file and atomic rename. Reject unsupported versions and parameter values above safe limits before allocating memory.

- [ ] **Step 5: Test add/update/clear, endpoint binding, backend fallback, file permissions, and redaction**

Run: `go test ./internal/credentials -v`  
Expected: PASS; the literal `canary-key` must not appear in JSON, errors, logs, or vault bytes.

- [ ] **Step 6: Commit**

```powershell
git add internal/credentials AGENT_LOG.md go.mod go.sum
git commit -m "feat: store provider credentials securely"
```

---

### Task 14: Application Service, Preflight, and Repository Locking

**Files:**
- Create: `internal/app/service.go`
- Create: `internal/app/preflight.go`
- Create: `internal/app/repolock.go`
- Create: `internal/app/composition_local.go`
- Test: `internal/app/service_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: config loader, store, loop factory, credentials, Git/workspace, clock.
- Produces: `app.Service.Preflight/CreateRun/GetRun/ListRuns/CancelRun/Approve/Reject`, `PreflightReport`, and one-active-run-per-repository locking.

- [ ] **Step 1: Write failing preflight and concurrency tests**

```go
func TestWorkspaceAutoRejectsDirtyRepository(t *testing.T) {
    repo := testrepo.New(t, map[string]string{"a.txt":"old"}); repo.Write("a.txt","dirty")
    got := app.NewTestService(t).Preflight(context.Background(), app.CreateRunRequest{RepoRoot:repo.Root, Profile:domain.ProfileWorkspaceAuto})
    if got.Code != "DIRTY_WORKTREE" { t.Fatalf("%#v", got) }
}
func TestOnlyOneActiveRunPerCanonicalRepository(t *testing.T) {
    svc := app.NewTestService(t); req := validRequest(t)
    if _, err := svc.CreateRun(context.Background(), req); err != nil { t.Fatal(err) }
    if _, err := svc.CreateRun(context.Background(), req); !errors.Is(err, app.ErrRepoBusy) { t.Fatalf("%v", err) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/app -v`  
Expected: FAIL because app package is missing.

- [ ] **Step 3: Implement preflight with distinct findings**

Check canonical Git root, Git executable, baseline commit/diff, config, platform stage resolution, executable lookup, key status, profile, dirty-worktree rule, and data-directory writability. Return ordered `Finding{Code, Severity, Message}` values; never include a secret or raw Authorization value.

- [ ] **Step 4: Implement use cases and canonical repository locks**

```go
type CreateRunRequest struct { RepoRoot, Task, Provider, Model, Endpoint string; Profile domain.PermissionProfile; ConfigPath string }
type PreflightReport struct { OK bool; Findings []Finding; BaselineCommit, BaselineDiffHash string }
type LoopController interface { Start(context.Context, domain.Run) error; Approve(context.Context, domain.RunID, string) error; Reject(context.Context, domain.RunID, string, bool) error; Cancel(context.Context, domain.RunID) error }
type Service struct { store store.Store; locks *RepoLocks; loops LoopController; creds *credentials.Service }
func (s *Service) Preflight(context.Context, CreateRunRequest) PreflightReport
func (s *Service) CreateRun(context.Context, CreateRunRequest) (domain.Run, error)
func (s *Service) Approve(context.Context, domain.RunID, string) error
func (s *Service) Reject(context.Context, domain.RunID, string, bool) error
func (s *Service) CancelRun(context.Context, domain.RunID) error
```

Release repository locks on every terminal state and on startup recovery of abandoned local runs. Never reset/clean/revert the repository.

- [ ] **Step 5: Run green and commit**

Run: `go test ./internal/app -v`  
Expected: PASS for clean/dirty profiles, missing Git/check/key, endpoint mismatch, same path via symlink, concurrent create, cancel, approve/reject, and lock release.

```powershell
git add internal/app AGENT_LOG.md
git commit -m "feat: orchestrate safe local harness runs"
```

---

### Task 15: Versioned HTTP API, SSE, and Local Web Security

**Files:**
- Create: `internal/httpapi/router.go`
- Create: `internal/httpapi/runs.go`
- Create: `internal/httpapi/approvals.go`
- Create: `internal/httpapi/credentials.go`
- Create: `internal/httpapi/events.go`
- Create: `internal/httpapi/security.go`
- Test: `internal/httpapi/api_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: `app.Service`, store event reads, credential service, route-capability set.
- Produces: `/api/v1` JSON/SSE API, local-session middleware, and a demo-safe route composition mechanism.

- [ ] **Step 1: Write failing route, origin, CSRF, SSE replay, and secret-response tests**

```go
func TestMutationRequiresSessionAndOrigin(t *testing.T) {
    api := newLocalAPI(t)
    req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", strings.NewReader(`{}`))
    req.Header.Set("Origin", "https://evil.example")
    rr := httptest.NewRecorder(); api.ServeHTTP(rr, req)
    if rr.Code != http.StatusForbidden { t.Fatalf("%d", rr.Code) }
}
func TestSSEReplaysAfterSequence(t *testing.T) {
    api, st := newLocalAPI(t)
    seedRunEvents(t, st, "run-1", "one", "two", "three")
    req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-1/events", nil)
    req.Header.Set("Last-Event-ID", "1")
    rr := httptest.NewRecorder(); api.ServeHTTP(rr, req)
    if strings.Contains(rr.Body.String(), "data: one") || !strings.Contains(rr.Body.String(), "data: two") || !strings.Contains(rr.Body.String(), "data: three") { t.Fatalf("%s", rr.Body.String()) }
}
func TestCredentialStatusResponseNeverContainsCanary(t *testing.T) {
    api := newLocalAPIWithCredential(t, "canary-key")
    req := authorizedRequest(t, http.MethodGet, "/api/v1/credentials/openai/api.openai.com", nil)
    rr := httptest.NewRecorder(); api.ServeHTTP(rr, req)
    if strings.Contains(rr.Body.String(), "canary-key") { t.Fatalf("leaked: %s", rr.Body.String()) }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/httpapi -v`  
Expected: FAIL because HTTP API is missing.

- [ ] **Step 3: Implement exact routes and JSON error envelope**

Routes:

```text
POST   /api/v1/runs
GET    /api/v1/runs/{id}
POST   /api/v1/runs/{id}/cancel
POST   /api/v1/runs/{id}/approvals/{digest}/approve
POST   /api/v1/runs/{id}/approvals/{digest}/reject
GET    /api/v1/runs/{id}/events
GET    /api/v1/runs/{id}/artifacts/{artifactID}
POST   /api/v1/config/validate
GET    /api/v1/credentials/{provider}/{host}
PUT    /api/v1/credentials/{provider}/{host}
DELETE /api/v1/credentials/{provider}/{host}
GET    /healthz
```

Errors use the exact envelope shape `{"error":{"code":"INVALID_JSON","message":"request body is not valid JSON","request_id":"req-123"}}`; messages are redacted and bounded.

- [ ] **Step 4: Implement local security and SSE**

Bind composition to `127.0.0.1`; generate a 32-byte random session token; exchange a one-time bootstrap token for an HttpOnly, SameSite=Strict cookie; redirect to a clean URL; require matching Host/Origin and a per-session CSRF header for mutations; emit `id`, `event`, and JSON `data` SSE fields with heartbeat comments and `Last-Event-ID` replay.

- [ ] **Step 5: Add demo route-capability tests and run green**

The router constructor accepts capabilities. A demo router must return 404 for credentials, arbitrary run creation, config validation, and artifacts outside fixed demo runs.

Run: `go test ./internal/httpapi -v`  
Expected: PASS for every route, invalid JSON, body size, auth/origin/CSRF, SSE replay/disconnect, 404 capability pruning, and secret canary.

- [ ] **Step 6: Commit**

```powershell
git add internal/httpapi AGENT_LOG.md
git commit -m "feat: expose secure local run API and events"
```

---

### Task 16: Open Design React WebUI

**Files:**
- Create: `DESIGN.md`
- Create: `web/package.json`
- Create: `web/package-lock.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/src/main.tsx`
- Create: `web/src/api/client.ts`
- Create: `web/src/api/types.ts`
- Create: `web/src/pages/Dashboard.tsx`
- Create: `web/src/pages/NewRun.tsx`
- Create: `web/src/pages/RunDetail.tsx`
- Create: `web/src/pages/Credentials.tsx`
- Create: `web/src/pages/DemoGallery.tsx`
- Create: `web/src/components/Timeline.tsx`
- Create: `web/src/components/ApprovalPanel.tsx`
- Create: `web/src/components/DiffViewer.tsx`
- Create: `web/src/styles/tokens.css`
- Test: `web/src/**/*.test.tsx`
- Test: `web/e2e/run.spec.ts`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: exact `/api/v1` schemas and SSE events from Task 15.
- Produces: embedded-build-ready SPA covering dashboard, new run, timeline/diff, approval, credentials, and simulated demos.

- [ ] **Step 1: Materialize the approved design system before components**

Use Open Design’s `dashboard` prototype skill with the `linear-app` system. Commit a nine-section `DESIGN.md` defining color, typography, spacing, layout, components, motion, voice, brand, and anti-patterns. Required visible rules: fixed light theme, status text plus icon (never color alone), `SIMULATED` badge for demo, keyboard focus ring, dense developer-tool layout, no decorative gradients, no embedded terminal.

- [ ] **Step 2: Create the Vite/Vitest/Testing Library scaffold and failing UI tests**

Run `npm create vite@8.1.0 web -- --template react-ts`, pin `react` and `react-dom` to 19.2, then install Vitest, Testing Library, Playwright, and axe-core as development dependencies. Commit the generated lockfile; subsequent CI uses `npm ci`.

Set Vite’s production `build.outDir` to `../internal/httpapi/webdist` with `emptyOutDir: true`. This generated directory remains ignored by Git and is rebuilt before every Go compilation once the embed package exists.

```tsx
it("renders validation failure evidence and remaining budgets", () => {
  render(<Timeline events={fixtureEvents} />)
  expect(screen.getByText("TEST_FAILURE")).toBeVisible()
  expect(screen.getByText("Mutations 1 / 5")).toBeVisible()
})

it("approval panel never offers a permanent allow", () => {
  render(<ApprovalPanel request={guardedPatch} onDecision={vi.fn()} />)
  expect(screen.getByRole("button", {name:"Approve once"})).toBeVisible()
  expect(screen.queryByText(/always allow/i)).toBeNull()
})
```

- [ ] **Step 3: Run red**

Run: `npm --prefix web test -- --run`  
Expected: FAIL because pages/components are absent.

- [ ] **Step 4: Implement typed client, pages, and components**

Keep server state in a small fetch/SSE client, not a global framework. Reconnect SSE using the latest sequence. Render raw redacted output only on explicit expand. Diff viewer is read-only. Credential inputs use password fields and clear local state immediately after submission. Public-demo mode hides unavailable controls based on server capabilities, while server-side route pruning remains authoritative.

- [ ] **Step 5: Add accessibility and browser E2E tests**

Browser scenario: open demo gallery → start feedback-loop scenario → observe policy denial → observe failed validation → observe changed second patch → reach succeeded → inspect final diff. Keyboard-only scenario must create a supervised local-form draft and open/close the approval panel. Run axe checks on all pages with zero serious/critical findings.

- [ ] **Step 6: Run green, build, and commit**

Run:

```powershell
npm --prefix web test -- --run
npm --prefix web run build
npm --prefix web run e2e
```

Expected: all PASS; compressed production assets remain below 1.5 MiB.

```powershell
git add DESIGN.md web AGENT_LOG.md
git commit -m "feat: add observable harness web interface"
```

---

### Task 17: CLI Composition Roots and Deterministic Mechanism Demo

**Files:**
- Create: `cmd/ai4se-harness/main.go`
- Create: `cmd/ai4se-harness/serve.go`
- Create: `cmd/ai4se-harness/run.go`
- Create: `cmd/ai4se-harness/credentials.go`
- Create: `cmd/ai4se-harness/demo.go`
- Create: `internal/demo/scenario.go`
- Create: `internal/demo/composition.go`
- Create: `internal/httpapi/web.go`
- Test: `internal/demo/scenario_test.go`
- Test: `cmd/ai4se-harness/main_test.go`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: local app/API composition, embedded `internal/httpapi/webdist`, mock provider/executor/store.
- Produces: `serve`, `run`, `credentials`, and `demo feedback-loop` commands plus a compile-time mock-only demo profile.

- [ ] **Step 1: Write the failing mechanism-demo test**

```go
func TestFeedbackLoopScenarioProvesRequiredMechanisms(t *testing.T) {
    result := demo.RunFeedbackLoop(context.Background())
    if result.Terminal != domain.StateSucceeded { t.Fatalf("%#v", result) }
    assertOrdered(t, result.Events,
        "PolicyDenied",
        "PatchApplied",
        "ValidationFailed",
        "FeedbackProduced",
        "PatchApplied",
        "ValidationPassed",
        "RunSucceeded",
    )
    if result.Actions[1].Digest == result.Actions[2].Digest { t.Fatal("feedback did not change action") }
}
```

- [ ] **Step 2: Run red**

Run: `go test ./internal/demo ./cmd/ai4se-harness -v`  
Expected: FAIL because commands/demo composition are missing.

- [ ] **Step 3: Implement CLI commands with hidden credential input**

```text
ai4se-harness serve --profile local --repo <path>
ai4se-harness run --repo <path> --task <text> --config .ai4se-harness.toml
ai4se-harness credentials set|status|clear --provider <id> --endpoint <url>
ai4se-harness demo feedback-loop --format text|json
```

Use `golang.org/x/term` for hidden terminal input. Do not accept a key flag or environment echo. `web.go` uses `//go:embed webdist/*` after Vite builds `internal/httpapi/webdist`; local opens a clean bootstrap URL, while demo binds the configured container address and exposes only mock routes.

- [ ] **Step 4: Implement fixed conditional demo scenario**

Decision sequence is feedback-dependent: initial raw-shell request → on `POLICY_DENIED`, incomplete patch → on `TEST_FAILURE`, corrected patch → `finish`. Mock executor/checker returns deterministic evidence. Public workspace is an in-memory map and cannot resolve host paths.

- [ ] **Step 5: Prove demo composition excludes real capabilities**

Add tests that inspect registered tools/routes/types and fail if `Local`, keyring/vault, credential routes, custom endpoints, or `os/exec`-backed executor are reachable. Run with network disabled in CI.

- [ ] **Step 6: Run green and commit**

Run:

```powershell
npm --prefix web run build
go test ./internal/demo ./cmd/ai4se-harness -v
go run ./cmd/ai4se-harness demo feedback-loop --format json
```

Expected: tests PASS; JSON terminal state is `SUCCEEDED` with ordered mechanism events.

```powershell
git add cmd internal/demo internal/httpapi/web.go AGENT_LOG.md go.mod go.sum
git commit -m "feat: ship local CLI and mock mechanism demo"
```

---

### Task 18: One-Command Tests, CI, Releases, Container, and Documentation

**Files:**
- Create: `scripts/test.ps1`
- Create: `scripts/test.sh`
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`
- Create: `Dockerfile`
- Create: `.dockerignore`
- Create: `deploy/compose.yml`
- Create: `deploy/Caddyfile`
- Create: `README.md`
- Create: `SECURITY.md`
- Create: `THIRD_PARTY_LICENSES.md`
- Create: `LICENSE`
- Test: `internal/demo/container_contract_test.go`
- Test: `internal/demo/workflow_contract_test.go`
- Modify: `PLAN.md`
- Modify: `AGENT_LOG.md`

**Interfaces:**
- Consumes: all prior build/test commands and the mock-only demo binary.
- Produces: one-command verification per target shell, GitHub Actions `unit-test`, Windows/Linux release artifacts, GHCR demo image, Ubuntu deployment files, and complete operator documentation.

- [ ] **Step 1: Write failing release/container contract tests**

Mark `container_contract_test.go` with `//go:build integration`. Test that the demo binary starts without writable root, serves `/healthz`, returns 404 for credential/custom-run routes, writes only beneath its tmpfs path, and exits cleanly on SIGTERM. In the untagged workflow contract test, parse workflow YAML and assert a job key exactly named `unit-test` with both `ubuntu-latest` and `windows-latest` matrix entries.

- [ ] **Step 2: Run red**

Run: `go test -tags=integration ./internal/demo -run 'Container|Workflow' -v`  
Expected: FAIL because workflows/container files are absent.

- [ ] **Step 3: Implement one-command developer verification**

`scripts/test.ps1` and `scripts/test.sh` both run, in order: frontend unit tests, frontend production build into `internal/httpapi/webdist`, `go test ./...`, `go vet ./...`, and browser E2E. They stop on first nonzero exit and never print environment variables. Document:

```powershell
./scripts/test.ps1
```

```bash
./scripts/test.sh
```

- [ ] **Step 4: Implement GitHub CI and release workflows**

`ci.yml` jobs: `unit-test` matrix (Windows/Ubuntu), `frontend-test`, `security-test`, `integration-test`, `build`, `docker-build`, and `e2e`. Every Go-compiling job runs `npm ci` and `npm run build` first so embedded assets exist. Pin action major versions and use least permissions. `release.yml` triggers on `v*` tags, builds `ai4se-harness_windows_amd64.exe` and `ai4se-harness_linux_amd64`, writes SHA-256 checksums, uploads GitHub Release assets, and pushes `ghcr.io/liu-ty/ai4se_coding_agent_harness:<tag>` plus immutable digest metadata.

- [ ] **Step 5: Build the mock-only production image and deployment config**

Multi-stage Dockerfile builds Node assets, then the Go demo binary, then copies only the binary and CA certificates into a non-root minimal image. Entrypoint is `serve --profile demo`. Compose sets read-only root, tmpfs, `cap_drop: [ALL]`, `no-new-privileges`, PID/memory/CPU limits, and loopback exposure to Caddy. Caddy uses `{$AI4SE_DOMAIN}` as the exact site label, obtains HTTPS automatically, and proxies only to demo.

- [ ] **Step 6: Write complete delivery documentation**

README sections exactly include: Overview, Why This Exists, Architecture, Installation, Windows Quickstart, Linux Quickstart, Configuration, Validation Pipeline, Permission Profiles, Credential Security, Running Locally, Public Demo, Distribution, Directory Structure, Security Boundaries, Known Limitations, Development, Testing, CI/CD, Deployment, Third-Party Licenses. `SECURITY.md` explains trusted-local-repository boundary and vulnerability reporting. `THIRD_PARTY_LICENSES.md` records every dependency and Open Design attribution. Project license is Apache-2.0.

- [ ] **Step 7: Run full verification and inspect fresh-machine artifacts**

Run:

```powershell
./scripts/test.ps1
go build -trimpath -o dist/ai4se-harness_windows_amd64.exe ./cmd/ai4se-harness
$env:GOOS='linux'; $env:GOARCH='amd64'; go build -trimpath -o dist/ai4se-harness_linux_amd64 ./cmd/ai4se-harness
docker build --target demo -t ai4se-harness-demo:test .
docker run --rm --read-only --tmpfs /tmp:rw,noexec,nosuid,size=64m ai4se-harness-demo:test demo feedback-loop --format json
```

Expected: all tests/builds PASS; container JSON says `SUCCEEDED`; `git grep` and secret scan find no real credentials.

- [ ] **Step 8: Commit and record final CI evidence**

```powershell
git add scripts .github Dockerfile .dockerignore deploy README.md SECURITY.md THIRD_PARTY_LICENSES.md LICENSE PLAN.md AGENT_LOG.md
git commit -m "build: package and verify course delivery"
```

After push, record the passing GitHub Actions run URL, release URL, GHCR digest, and deployed HTTPS URL in `README.md` and `AGENT_LOG.md` in a follow-up documentation commit.

---

## Human-Only Course Deliverables

The student, not an implementation agent, must:

1. Write `REFLECTION.md` in their own words (1500–2500 Chinese characters/words as required by the course wording).
2. Disclose any AI copy-editing of that reflection.
3. Provision DNS and server access, review `deploy/compose.yml`/`Caddyfile`, and perform the production deployment.
4. Verify the final public URL and final GitHub Actions run before submission.
5. Review every PR, record human modifications, and accept responsibility for the delivered code.

## Dependency and Parallelism Graph

```text
Cold-start gate
  └─ Task 1
      ├─ Tasks 2 ─┬─ Task 8 ─ Task 9 ─┐
      ├─ Task 3   │                   │
      ├─ Task 4   └─ Task 10 ─────────┤
      ├─ Task 5 ─ Task 7 ─────────────┤
      └─ Task 6 ─ Task 7 ─────────────┤
                                      └─ Task 11
                                          ├─ Task 12 ─┐
                                          └─ Task 13 ─┴─ Task 14 ─ Task 15 ─ Task 16 ─ Task 17 ─ Task 18
```

Tasks 2–4, 5–7, and 8–10 use separate worktrees after Task 1. Tasks 12 and 13 may run concurrently after Task 11 freezes the provider/credential contracts.

## Spec Traceability Matrix

| SPEC requirement | Plan task(s) |
|---|---|
| Self-built state machine and agent loop | 1, 11 |
| Provider-neutral mock/OpenAI/Anthropic | 11, 12 |
| Decision/tool/config dimensions | 1, 2, 5–7 |
| Cross-platform restricted executor | 8 |
| Ordered objective validation | 9, 11 |
| Deep feedback classification/fingerprints/progress | 4, 10, 11 |
| Review/supervised/workspace-auto and HITL | 5, 11, 15–16 |
| Run-scoped memory and immutable events | 3, 11 |
| Secure keyring/vault and endpoint binding | 12–15 |
| Local API/SSE and embedded WebUI | 15–17 |
| Public mock-only mechanism demo | 11, 17–18 |
| Windows/Linux binaries and GHCR image | 8, 17–18 |
| One-command tests and GitHub `unit-test` CI | 18 |
| Required README/security/distribution documentation | 18 |
| A.6 guardrail + injected failure + changed action | 11, 17 |

## Plan Completion Definition

The plan is complete only when:

- Every task checkbox is checked and annotated with its commit hash.
- Every mapped worktree has a reviewed PR and all critical findings are resolved.
- `SPEC_PROCESS.md` contains cold-start evidence and resulting diffs.
- `AGENT_LOG.md` contains prompts/context, subagent identity, output/commit, human intervention, and lessons for every task.
- The latest GitHub Actions run passes.
- GitHub Release and GHCR artifacts exist and match checksums/digests.
- The public HTTPS mock WebUI is reachable and exposes no real execution/credential surface.
- The student-authored `REFLECTION.md` is present.
