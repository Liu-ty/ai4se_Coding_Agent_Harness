# Agent Log

This log records the AI-assisted engineering process. It is append-only by convention. No real credential or unredacted secret may be recorded.

## 2026-07-10 — Requirements Review

- **Task:** PRE-001
- **Skills:** `superpowers:using-superpowers`
- **Context:** Read the course-wide project requirements and the Coding Agent Harness-specific requirements as one combined specification.
- **Key output:** Identified mandatory Superpowers workflow, TDD, mock-LLM deterministic tests, mechanism demo, secure credentials, distribution, CI, WebUI, and the prohibition on reusing an existing high-level agent loop.
- **Human intervention:** User confirmed that GitHub, not GitLab, is the authoritative repository/CI target.
- **Lesson:** Course scoring emphasizes an independently testable mechanism and process evidence more than breadth or line count.

## 2026-07-12 15:00 +08:00 — Brainstorming Started

- **Task:** SPEC-001
- **Skills:** `superpowers:brainstorming`
- **Context:** Empty workspace; no existing code, documentation, or Git history.
- **Key decisions:** Feedback loop as primary contribution; language-agnostic checks; local real execution plus public mock demo.
- **Human intervention:** User supplied the server constraint: 2 CPU, 2 GB RAM, Ubuntu, with a domain.
- **Lesson:** Hosting constraints favor a mock-only public composition rather than insecure low-resource cloud code execution.

## 2026-07-12 15:15 +08:00 — Technology and Provider Decisions

- **Task:** SPEC-002
- **Skills:** `superpowers:brainstorming`, `openai-docs`
- **Context:** Compared Python/FastAPI, TypeScript/Node, and Go; compared OpenAI-compatible, Anthropic, and full streaming/tool parity.
- **Key output:** Go 1.26 core, React/TypeScript WebUI, provider-neutral JSON decisions, mock/OpenAI-compatible/Anthropic adapters, no token streaming.
- **Human intervention:** User accepted Go + React and the limited multi-provider compatibility scope.
- **Lesson:** Provider compatibility is affordable only when the harness owns a small canonical decision protocol.

## 2026-07-12 15:25 +08:00 — Editing, Task Scope, and Extensibility

- **Task:** SPEC-003
- **Skills:** `superpowers:brainstorming`, `openai-docs`
- **Context:** Compared patch, whole-file, and shell editing against mainstream coding-agent patterns.
- **Key output:** Patch-first actions, configured checks, defect-first MVP, acceptance-driven small features for delivery, ports/adapters roadmap.
- **Human intervention:** User rejected arbitrary task breadth and confirmed the bounded scope plus future complete-agent evolution.
- **Lesson:** Extensibility should be expressed by stable seams and events, not by an unimplemented plugin platform.

## 2026-07-12 15:35 +08:00 — Governance, Platform, and Distribution

- **Task:** SPEC-004
- **Skills:** `superpowers:brainstorming`
- **Key output:** Review/supervised/workspace-auto profiles; Windows x64 and Linux x64; GitHub Releases plus GHCR; keyring plus encrypted vault; staged validation; dual budgets.
- **Human intervention:** User confirmed every selection.
- **Lesson:** Permission modes may change approval friction, but hard security denials must remain profile-independent.

## 2026-07-12 15:45 +08:00 — Design Review

- **Task:** SPEC-005
- **Skills:** `superpowers:brainstorming`
- **Context:** Presented architecture, feedback state machine, tool/security boundaries, data/API/UI, testing/delivery, and product/risk contract in six sections.
- **Key output:** All six design sections approved by the user.
- **Human intervention:** Project branding was deferred; course name retained.
- **Lesson:** The resulting product is a complete minimal harness with one deep dimension, not a partial imitation of a commercial agent.

## 2026-07-12 15:53 +08:00 — Specification Written

- **Task:** SPEC-006
- **Skills:** `superpowers:brainstorming`
- **Context:** Initialized the local Git repository and materialized the approved design.
- **Key output:** `SPEC.md`, `SPEC_PROCESS.md`, and `AGENT_LOG.md`.
- **Human intervention:** Written specification still requires explicit user review before `PLAN.md` may be generated.
- **Lesson:** Planning documents are project artifacts and process evidence; implementation remains prohibited until plan and cold-start validation are complete.

## 2026-07-12 16:00 +08:00 — GitHub Repository Bound

- **Task:** SPEC-007
- **Skills:** `superpowers:brainstorming`
- **Context:** User supplied the authoritative repository URL.
- **Key output:** Git remote and module identity fixed to `git@github.com:Liu-ty/ai4se_Coding_Agent_Harness.git` and `github.com/Liu-ty/ai4se_Coding_Agent_Harness`.
- **Human intervention:** Repository owner and exact casing were supplied by the user.
- **Lesson:** Repository identity must be fixed before an implementation plan specifies module imports, release artifacts, and CI references.

## 2026-07-12 16:05 +08:00 — Written SPEC Approved

- **Task:** PLAN-000
- **Skills:** `superpowers:brainstorming`
- **Context:** User reviewed the materialized root-level specification after commit `137fe2f`.
- **Key output:** User explicitly approved `SPEC.md`, satisfying the brainstorming written-review gate.
- **Human intervention:** Approval was provided as “确认”.
- **Lesson:** Conversational section approval and written-artifact approval are separate evidence points.

## 2026-07-12 16:15 +08:00 — Implementation Plan Written

- **Task:** PLAN-001
- **Skills:** `superpowers:writing-plans`
- **Context:** Decomposed the approved specification without creating implementation code.
- **Key output:** Root `PLAN.md` with 18 TDD tasks, exact interfaces/files/commands, eight worktree/PR groups, cold-start gate, traceability matrix, and human-only delivery boundaries.
- **Human intervention:** None after written SPEC approval; implementation plan awaits user review.
- **Lesson:** Cross-platform process control, public-demo route pruning, and credential fallback require explicit test tasks rather than being left to final packaging.

## 2026-07-21 — Repository Recovery After Storage Loss

- **Task:** RECOVERY-001
- **Skills:** `superpowers:systematic-debugging`, `superpowers:writing-plans`, `superpowers:verification-before-completion`.
- **Context:** The local repository was missing and the GitHub repository had no remote refs. A surviving local Git-object snapshot contained the approved planning-document baseline.
- **Key output:** Recovered the exact four-document baseline from Git blobs, rebuilt `main`, restored `origin`, and committed the baseline as `87482f1` (`docs: restore project planning baseline`). Original commit hashes could not be recreated because their commit objects were lost.
- **Evidence:** Recovery provenance and exact blob IDs are recorded in `SPEC_PROCESS.md`; the student's prior implementation-plan approval is restored, and the strengthened Task 2 plan now makes every required semantic validator and both review stages explicit.
- **Human intervention:** The user reported the storage loss and authorized reconstruction from project history and requirements.
- **Lesson:** Local-only artifacts are not substitutes for pushing Git history; future approved checkpoints should be pushed after review so a disk loss cannot erase the audit trail.

## 2026-07-21 — Restored Baseline Published to GitHub

- **Task:** RECOVERY-002
- **Skills:** `superpowers:using-superpowers`, `superpowers:writing-plans`.
- **Context:** Verified a clean `main`, an empty GitHub branch namespace, the configured SSH remote, and authenticated read access before publication.
- **Key output:** Published recovery head `f81de6d` to `origin/main` and configured the local branch to track it. No implementation or experiment branch was published.
- **Human intervention:** The user explicitly requested continued progress after reviewing the reconstruction result.
- **Lesson:** A reviewed documentation gate should be pushed before starting disposable validation worktrees so the authoritative baseline remains independently recoverable.

## 2026-07-22 20:10 +08:00 — Cold Start Experiment: Task 1 & Task 2

- **Task:** COLD-START-001
- **Skills:** `superpowers:using-superpowers`, `superpowers:using-git-worktrees`, `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:requesting-code-review`, `superpowers:verification-before-completion`
- **Context:** Controlled cold-start experiment. Only committed `SPEC.md` and `PLAN.md` provided. No conversation history, no external research. Executed on isolated worktree `cold-start/opencode-clean-20260722-rerun` at `dee5d29`.
- **Key output:**
  - Task 1: Project skeleton (`.gitattributes`, `.gitignore`, `go.mod`) and run state machine (`internal/domain/types.go`, `internal/domain/state.go`, `internal/domain/state_test.go`). All 10 tests pass.
  - Task 2: Strict versioned project configuration (`internal/config/config.go`, `internal/config/load.go`, `internal/config/resolve.go`, `internal/config/config_test.go`, `internal/config/config_windows_test.go`, `testdata/config/valid.toml`). All 22 tests pass.
  - Dependency: `github.com/BurntSushi/toml` v1.6.0 added via `go mod tidy`.
- **Red-green evidence:**
  - Task 1 red: `go test ./internal/domain` → `FAIL: no non-test Go files`
  - Task 1 green: `go test ./internal/domain -v` → PASS (3 tests)
  - Task 2 red: `go test ./internal/config` → `FAIL: no non-test Go files`
  - Task 2 green: `go test ./internal/config -v` → PASS (17 tests initially, 22 after review fixes)
- **Spec compliance review:** Oracle agent (ses_07639c92fffedn03BdXkDDAvbZ) — 73/73 checks PASS. Minor observations: Load returns `*Config` not `Config` (idiomatic), ResolveStage uses `goos string` not `runtime.GOOS` (more flexible). No functional defects.
- **Code quality review:** Oracle agent (ses_076370f28ffej5Cb4XAg1f3DAA) — 0 CRITICAL, 6 IMPORTANT, 10 MINOR. All IMPORTANT issues fixed:
  - Added explicit terminal state entries to transition map
  - Added `default_profile` validation with `ErrInvalidProfile` sentinel
  - Added `Classifiers` field to `CommandSpec`
  - Added `//go:build windows` tag to platform-specific test
  - Added empty working directory validation with `ErrEmptyWorkingDirectory` sentinel
  - Added tests for terminal states, full repair flow, invalid timeout, Classifiers preservation, darwin fallback
  - Moved transition map to package-level variable
  - Fixed error wrapping consistency (`%v` → `%w`)
- **Final verification:** `go test ./...` → PASS (2 packages), `git diff --check` → clean.
- **Human intervention:** None. No commits performed per experiment rules.
- **Lesson:** Platform-specific test files require correct Go naming conventions (`*_windows_test.go` not `*_test_windows.go`). Plan-level test stubs may omit fields needed for implementation validation (e.g., `Timeout` in `ValidationStage`). Review feedback should be addressed systematically before claiming completion.

## 2026-07-23 — Independent Cold-Start Remediation

- **Task:** COLD-START-VERIFY-001
- **Skills:** `superpowers:using-superpowers`, `superpowers:receiving-code-review`, `superpowers:systematic-debugging`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Context:** Independently verified the OpenCode result under a student-approved moderate acceptance standard: retain useful Task 1/2 code, disclose non-destructive process deviations, and require security, public-interface, formatting, and evidence corrections without repeating the full cold start.
- **Corrections:**
  - Added a compile-time `Load(io.Reader) (Config, error)` contract, observed the expected pointer/value red failure, and changed `Load` to return `Config`.
  - Added platform-neutral Windows drive, UNC, root-relative, and POSIX absolute-path cases, observed the expected `\rooted` red failure, and implemented host-independent rejection.
  - Removed the Windows-only path test, formatted all Go files, removed `.omo` temporary state, and corrected `PLAN.md`, `SPEC_PROCESS.md`, and `COLD_START_REPORT.md`.
- **Accepted deviations:** The original session did not explicitly invoke `using-superpowers`; review-fix tests lacked red-state evidence; reviewer inputs were file snapshots rather than a complete Git diff. These waivers apply only to this cold-start gate.
- **Fresh verification:** `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, and Linux x64 cross-builds all passed. Task 3 and later tasks remain prohibited until this evidence is reviewed and committed.
- **Human intervention:** The student explicitly approved conditional acceptance and execution of the remediation plan.
- **Lesson:** Cold-start success is determined by independent evidence and honest remediation, not the worker agent's self-reported terminal label.

## 2026-07-23 — Cold-Start Integration and Task 1 Closure

- **Task:** FOUNDATION-INTEGRATION-001
- **Skills:** `superpowers:using-superpowers`, `superpowers:executing-plans`, `superpowers:receiving-code-review`, `superpowers:using-git-worktrees`, `superpowers:verification-before-completion`.
- **Context:** The accepted cold-start result had been separated locally into evidence commit `496587a`, Task 1 commit `c76bfd8`, and a dependent Task 2 branch, but `PLAN.md` still described the evidence commit as pending and retained unchecked Task 1 steps.
- **Key output:** Closed the cold-start evidence checkbox, annotated Task 1 with implementation commit `c76bfd8`, marked its five planned steps complete, and added the durable Codex session roadmap under `docs/superpowers/plans/`.
- **Verification:** On the `foundation` branch, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, Linux amd64 package build, and Linux amd64 test-binary compilation all passed.
- **Human intervention:** The student authorized execution of the recommended process closure, branch publication, and Foundation PR preparation. No merge was authorized.
- **Lesson:** Splitting implementation commits is not sufficient by itself; the plan status, task hash, checkboxes, and durable agent log must describe the same Git state before publication.

## 2026-07-23 — Task 2 Branch Integration Closure

- **Task:** CONFIG-INTEGRATION-001
- **Skills:** `superpowers:using-superpowers`, `superpowers:executing-plans`, `superpowers:receiving-code-review`, `superpowers:using-git-worktrees`, `superpowers:verification-before-completion`.
- **Context:** After the Foundation process commit, the unpublished `config-store-budget` branch was rebased onto `foundation`, changing the Task 2 implementation commit from `8a2e9bb` to `0d244ea` without changing its patch.
- **Key output:** Annotated Task 2 with implementation commit `0d244ea`, marked its seven planned steps complete, and updated the plan status to state that Tasks 3–4 have not started.
- **Verification:** On `config-store-budget`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, Linux amd64 package build, and Linux amd64 test-binary compilation all passed. The first sandboxed attempt could not read the external Go build cache; the identical commands passed outside the restricted sandbox.
- **Human intervention:** The student authorized the process repair and remote backup/PR preparation; Task 3 implementation and PR merge remain outside this step.
- **Lesson:** Record the post-rebase commit identifier, not the obsolete pre-rebase hash, and distinguish an environment permission failure from a failing test assertion.

## 2026-07-23 — Independent Review Fix and Final Branch Alignment

- **Task:** CONFIG-REVIEW-FIX-001
- **Skills:** `superpowers:requesting-code-review`, `superpowers:receiving-code-review`, `superpowers:systematic-debugging`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Context:** An independent GPT-5.6 Sol review accepted the Foundation range with documentation fixes and found that `C:relative` could bypass the host-independent working-directory guard on Windows. A second bounded reviewer confirmed the commit layering and Task 3 boundary but did not identify that path case.
- **Red-green evidence:** Added `C:relative` and `c:relative` to `TestLoadRejectsCrossPlatformAbsoluteWorkingDirectories`; both failed with `got <nil>` before the fix and passed after all leading ASCII drive designators were rejected.
- **Key output:** Moved the two evidence-text corrections into Foundation commit `5dfe967`; rebased Task 2 to implementation commit `83c03a2`; retained the TDD review fix as `624f6b9`; and kept Task 3 unstarted.
- **Verification:** After the final rebase, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, Linux amd64 package build, and Linux amd64 test-binary compilation all passed.
- **Human intervention:** The student authorized execution and publication preparation but did not authorize merging either branch.
- **Lesson:** Cross-host path validation must reject drive-relative forms as well as drive-rooted forms; a fast release-gate review does not replace a deeper security-focused review.

## 2026-07-24 - Task 3: Hash-Chained Memory and SQLite Event Store

- **Task:** Task 3 (implementation only; PLAN.md remains unmarked pending controller review).
- **Skills:** `superpowers:test-driven-development`.
- **Red evidence:** `go test ./internal/store -v` initially failed with `no non-test Go files in F:\\codes\\ai4se\\internal\\store`, after the contract test was added and before production store code existed. A restricted-sandbox attempt first failed only because the external Go build cache was inaccessible; the same command outside that restriction produced the expected red result.
- **Green evidence:** `go test ./internal/store -v` passed the memory and SQLite contracts, canonical hash verification, ID/type sentinels, payload isolation, concurrent sequence allocation, UpdateRun failure atomicity, empty content handling, and SQLite close/reopen persistence. `go test ./...` passed for `config`, `domain`, and `store`; `git diff --check` was clean.
- **Implementation:** Added domain events/artifacts, a concurrency-safe memory store, a SQLite store using `modernc.org/sqlite`, embedded schema migration, length-prefixed SHA-256 event hashes, stable input-validation sentinels, transaction-backed updates, and SQLite foreign-key/busy-timeout/WAL/single-writer configuration.
- **Self-review:** Corrected SQLite zero-time round-tripping and empty BLOB handling after focused tests exposed them. Verified the hash field order is run ID, sequence, type, Unix nanoseconds, payload, previous hash, each with an eight-byte big-endian length prefix.
- **Human intervention:** Approved normal dependency-download and Go build-cache access prompts; no product-design decisions were required.
- **Lesson:** Store contract tests must cover zero-value domain fields and empty byte slices, not only typical populated data.

## 2026-07-24 — Task 3 Independent Review and Closure

- **Task:** TASK-3-REVIEW-001.
- **Skills:** `superpowers:subagent-driven-development`, `superpowers:requesting-code-review`, `superpowers:receiving-code-review`, `superpowers:verification-before-completion`.
- **Context:** GitHub showed Foundation PR #1 already merged at `6cad5d1`. `main` was merged into `config-store-budget` without rewriting history (`116b106`), Task 2 was reverified on Windows and by Linux amd64 cross-compilation, and the synchronized branch was pushed before Task 3 began. Because the existing Task 2 worktree was outside the active writable sandbox, Task 3 was implemented on temporary branch `codex/task3-store` from the same `116b106` base and is intended to fast-forward `config-store-budget` after verification.
- **Implementation commits:** `b7814ee` added the hash-chained memory/SQLite event store; `110cbf6` added successful `UpdateRun` and post-snapshot event-insert rollback evidence; `01891d0` aligned duplicate-run, missing-parent artifact, timestamp, empty-list, large-sequence, and per-connection SQLite PRAGMA semantics.
- **Spec review:** An independent GPT-5.6 Sol reviewer initially found that the rollback test stopped before entering the transaction. After the trigger-based SQLite regression test and common successful `UpdateRun` contract were added, the reviewer returned **Approved** with no Critical, Important, or Minor findings.
- **Quality review:** A different independent GPT-5.6 Sol reviewer found backend error mismatches and connection-scoped PRAGMAs that could be lost when `database/sql` replaced a physical connection. After stable sentinel parity, DSN-scoped PRAGMAs, WAL verification, connection-replacement coverage, timestamp normalization, full concurrent hash-chain validation, a fixed digest vector, and large-sequence handling were added, the reviewer returned **Approved** with no remaining findings.
- **Verification:** Focused store tests, `go test -race ./internal/store -count=1`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l internal`, and `git diff --check` passed on the reviewed `01891d0` code. The controller still runs the full common exit gate and Linux amd64 cross-compilation before publication.
- **Human intervention:** The user authorized autonomous progress from the supplied project requirements and checklist. Approval prompts were limited to Git/GitHub synchronization, dependency download, and access to the existing Go build cache; no product behavior was selected by the user during Task 3.
- **Lesson:** A shared store interface is not proven interchangeable by happy-path contracts alone. Duplicate IDs, foreign-key failures, connection replacement, time normalization, and extreme sequence bounds must have backend-neutral observable semantics.

## 2026-07-24 - Task 4: Dual Budgets and No-Progress Detection

- **Task:** Task 4 implementation only; `PLAN.md` remains unchanged for controller review.
- **Skills:** `superpowers:test-driven-development`.
- **Red evidence:** Created the `internal/budget` boundary tests before production code. `go test ./internal/budget -v` then failed as expected because `NewProgressDetector`, `Limits`, `Tracker`, `Usage`, and the budget sentinels did not yet exist.
- **Green evidence:** After the minimal implementation, `go test ./internal/budget -v` passed all 12 tracker/progress tests. `go test -race ./internal/budget -count=1`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l internal`, and `git diff --check` all passed.
- **Implementation:** Added a mutex-protected tracker with independent decision, mutation, protocol-repair, and wall-clock limits; exported sentinel errors plus `StopError.Reason`; value snapshots; and a consecutive pair-based progress detector that resets on incomplete or changed signals.
- **Self-review:** Count records compare before incrementing, so attempt N+1 is rejected without usage inflation. The fake-clock boundary test verifies `elapsed >= WallClock`, while failed time checks leave usage intact. Detector tests verify both members of the pair must repeat, threshold one works, and empty evidence resets state.
- **Concerns:** None identified. `New` intentionally requires a non-nil caller-supplied `Clock`, matching the defined constructor contract and allowing deterministic harness integration.

## 2026-07-24 — Task 4 Independent Review and Closure

- **Task:** TASK-4-REVIEW-001.
- **Skills:** `superpowers:subagent-driven-development`, `superpowers:requesting-code-review`, `superpowers:receiving-code-review`, `superpowers:verification-before-completion`.
- **Implementation commits:** `c381b15` added decision, mutation, protocol-repair, and wall-clock budgets plus consecutive failure/diff detection; `d015fb5` made stop errors immutable, defined the nil-clock contract, removed the injected-clock callback from the tracker lock, and added shared-state race coverage.
- **Spec review:** An independent GPT-5.6 Sol reviewer returned **Approved** with no findings. Exact limits, typed stop reasons, fake-clock boundaries, and pair-based no-progress resets matched the task brief.
- **Quality review:** A different independent GPT-5.6 Sol reviewer initially found mutable stop reasons, an implicit nil-clock panic, and a re-entrant clock deadlock risk. After the review fix, the reviewer returned **Approved** with no Critical or Important findings. One Minor remains recorded for final branch review: the re-entrant regression test uses a 200 ms timeout that could be tight on an unusually slow CI worker.
- **Verification:** Focused tests, `go test -race ./internal/budget -count=1`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l internal`, and `git diff --check` passed on `d015fb5`.
- **Human intervention:** None beyond the earlier authorization to continue autonomous project progress.
- **Lesson:** Dependency injection boundaries must not call external callbacks while holding internal locks, and typed errors must not expose mutable state that can contradict their sentinel identity.

## 2026-07-24 — Course Requirements and Task 4 Plan Reconciliation

- **Task:** TASK-4-SPEC-RECONCILIATION-001.
- **Skills:** `superpowers:using-superpowers`, `superpowers:receiving-code-review`, `superpowers:writing-plans`.
- **Context:** The final Tasks 2–4 review was checked against both course requirement documents and the approved `SPEC.md` before accepting or rejecting each finding.
- **Decision:** Confirmed three Task 4 boundary defects: missing configuration defaults/validation for SPEC §4.7, run-global rather than per-decision protocol repairs, and a Boolean progress API unable to emit the required changed-diff warning. Confirmed three separate Task 3 architecture defects but deferred them to a post-pause integration gate.
- **Key output:** Amended `PLAN.md` with explicit failing tests, minimal implementation behavior, fresh two-stage review, and a hard pause after Task 4. Recorded that `SPEC.md` itself remains authoritative and unchanged.
- **Human intervention:** The student requested SPEC confirmation, plan correction, completion of Task 4, and a pause immediately afterward.
- **Lesson:** A prior task-level approval does not override a later whole-branch finding when the approved SPEC states a stricter observable contract; remediation scope should follow task ownership rather than mixing architecture work into an unrelated task.

## 2026-07-24 - Task 4 SPEC-Alignment Remediation

- **Task:** TASK-4-SPEC-ALIGNMENT-001 (implementation phase; review and completion markers remain pending).
- **Skill:** `superpowers:test-driven-development`.
- **Scope:** Task 2 Step 8 and Task 4 Steps 5-7 only: configured budget defaults/validation, per-decision-point protocol-repair allowance, typed progress outcomes, and the re-entrant clock timeout correction.
- **Red evidence:** After adding the focused tests and before modifying production code, `go test ./internal/config ./internal/budget -count=1` failed. The config package reported `undefined: config.ErrInvalidBudget`; the budget package reported `undefined: ProgressOutcome`, `ProgressContinue`, `ProgressWarning`, and `ProgressStop`. These failures directly identify the missing SPEC-aligned contracts rather than a test setup or environment failure.
- **Green evidence:** The focused config/budget command passed after the minimum implementation. `go test -race ./internal/config ./internal/budget -count=1`, `go test ./... -count=1`, `go vet ./...`, `gofmt -l internal`, `git diff --check`, and a `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` cross-compilation also passed.
- **Implementation:** `config.Load` now decodes into the four SPEC defaults and rejects every resolved non-positive count or malformed/non-positive wall-clock value with `ErrInvalidBudget`. A successful `Tracker.RecordDecision` resets only the current decision point's protocol-repair usage. `ProgressDetector.Observe` now returns the typed outcomes `continue`, `warning`, and `stop`, with a warning for the same failure plus a changed diff. The re-entrant clock test timeout is 2 seconds.
- **Review hardening:** The first fresh SPEC review approved the diff with no findings. The fresh quality review found that the already-correct rejected-decision path lacked regression coverage, so a test now proves a failed `RecordDecision` neither clears current protocol-repair usage nor reopens an exhausted allowance. The review also prompted clarification that `Usage.ProtocolRepairs` is current-decision-point usage and a broader budget sentinel message that covers malformed durations.
- **Scope verification:** Only `internal/config`, `internal/budget`, and this log changed. Task 3 store code, `go.mod`, `go.sum`, Task 5, and PLAN completion markers were not modified.
- **Human intervention:** The student confirmed the SPEC reconciliation, requested the plan update, and required a pause after Task 4.
- **Concern:** None within the Task 4 scope; the separately documented Task 3 architecture gate remains intentionally deferred.

## 2026-07-24 — Task 4 SPEC-Alignment Review and Pause

- **Task:** TASK-4-SPEC-ALIGNMENT-REVIEW-001.
- **Skills:** `superpowers:subagent-driven-development`, `superpowers:requesting-code-review`, `superpowers:receiving-code-review`, `superpowers:verification-before-completion`.
- **Implementation commit:** `994530d` (`fix: align Task 4 budget semantics with spec`).
- **SPEC review:** A fresh reviewer compared the diff with the course documents, `SPEC.md` §4.7/§6.1, and the corrected plan. Result: **Approved**, with no Critical, Important, or Minor findings.
- **Quality review:** A different fresh reviewer requested one Important regression test proving a rejected decision cannot reset or reopen the current protocol-repair allowance, plus two documentation/error-message clarifications. After those corrections, re-review result: **Approved**, with no remaining findings.
- **Verification:** The controller's fresh final gate passed: `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, and Linux amd64 CGO-disabled `go build ./...`.
- **Pause boundary:** Task 4 is complete. The confirmed Task 3 architecture issues remain unchecked in the post-Task-4 integration gate; no Task 3 remediation, Task 5 implementation, dependency tidy, branch publication, or PR-readiness claim was performed.
- **Lesson:** Per-decision counters need tests for both successful boundary transitions and rejected transitions; otherwise a future reset-before-validation regression can silently turn a bounded repair loop into an unbounded one.

## 2026-07-27 — Post-Task-4 Store Architecture Gate

- **Task:** CONFIG-STORE-BUDGET-INTEGRATION-GATE-001.
- **Skills:** `superpowers:using-superpowers`, `superpowers:brainstorming` (resume context only; prior approved SPEC/PLAN satisfied the design gate), `superpowers:executing-plans`, `superpowers:using-git-worktrees`, `superpowers:test-driven-development`, `superpowers:systematic-debugging`, `superpowers:requesting-code-review`, `superpowers:receiving-code-review`, `superpowers:verification-before-completion`.
- **Context:** Resumed after the explicit Task 4 pause. `PLAN.md` required the deferred Task 3 architecture gate before declaring `config-store-budget` PR-ready or starting Task 5.
- **Red evidence:** Added tests first for the new `internal/storeport` boundary, already-cancelled context parity, and injected store clocks. Focused RED command:
  `go test ./internal/storeport ./internal/store -run 'TestStorePort|TestStoresHonorAlreadyCancelledContextsWithoutMutation|TestStoresUseInjectedClockForEventTimestampsAndHashes' -v`
  failed because `internal/storeport` had no non-test Go files and `internal/store` could not compile after tests were migrated to `storeport.Store`.
- **Implementation:** Added `internal/storeport` with the `Store` port and shared sentinel errors; kept memory/SQLite adapters in `internal/store`; changed both stores to use constructor-level clocks for event creation; made every public store operation return an already-cancelled context before mutation; moved store errors to `storeport`; and ran `go mod tidy`, which made `modernc.org/sqlite` a direct dependency.
- **Green evidence:** The same focused command passed for `storeport` and both store implementations. Full local gate passed with exit code 0:
  `go test ./... -count=1; go test -race ./... -count=1; go vet ./...; go mod verify; gofmt -l internal; git diff --check`.
  Linux amd64 cross gate also passed with exit code 0: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...`, followed by `go test -c` for all packages into `.superpowers/sdd/cross`.
- **Environment note:** Sandboxed Go 1.26.5 repeatedly printed `error acquiring upload token` for `C:\Users\12155\AppData\Roaming\go\telemetry\local\upload.token` even when tests/builds exited 0. `GOPATH`, `GOCACHE`, and `GOMODCACHE` were redirected to `.superpowers/sdd` so module downloads and checksum state stayed inside the workspace; the telemetry warning did not correspond to a test, vet, module, or build failure.
- **Reviews:** Fresh SPEC reviewer `019fa19b-7b7d-7bd3-a12a-5d1f34ee1f58` found no Critical issues and one Important documentation issue: `PLAN.md` still pointed future consumers at `store.Store`. Fresh code-quality reviewer `019fa19b-b267-71c2-8c49-f5fce5d5ea66` found no Critical code blockers and the same Important plan/staging issue. `PLAN.md` now includes `internal/storeport/`, updates Task 3 and Task 14 snippets to `storeport.Store`, checks the integration-gate items, and records `internal/storeport` in the planned stage command.
- **Minor review notes deferred:** The nil-clock panic message is not separately locked by a test, `Clock.Now()` is still expected to be a simple clock and is called while memory holds its mutex / SQLite holds a transaction, and the injected-clock test checks returned events rather than reloading them via `ListEvents`. These were classified as Minor/Nice-to-have and did not block the gate.
- **Human intervention:** The user asked to continue from the documents and Superpowers. No product behavior beyond the already-approved SPEC/PLAN was selected during this gate.
- **Lesson:** Architecture gates need plan repair as much as code repair; otherwise future tasks can accidentally rebuild against the concrete adapter package and undo the intended dependency boundary.

## 2026-07-27 - PR #2 CodeRabbit Follow-up

- **Task:** PR-2-CODERABBIT-FOLLOWUP-001.
- **Skills:** `superpowers:receiving-code-review`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`, `github:gh-address-comments`.
- **Review intake:** CodeRabbit reported five actionable comments and three nitpicks on PR #2. Each finding was checked against the current `config-store-budget` diff before changing code.
- **Red evidence:** Added `TestSQLiteDSNNormalizesRelativePaths` before changing `sqliteDSN`; it failed because `sqliteDSN("state/runs.db")` produced `file://state/runs.db?...` with host `state`.
- **Implementation:** Replaced SQLite setup and trigger calls with context-aware database methods; normalized relative SQLite paths to absolute file URIs while preserving DSN pragmas; propagated `ListEvents` `rows.Close` errors; replaced deprecated TOML decoding; documented the single-connection hash-chain dependency and lock-free `CheckTime` invariant; reconciled `SPEC_PROCESS.md` with the completed deferred gate in `PLAN.md`.
- **Verification:** Focused config/budget/store tests, full `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, Linux amd64 `go build ./...`, and Linux amd64 test-binary compilation all exited 0. The Go tool still printed the pre-existing telemetry upload-token warning on this Windows machine.
- **GitHub writes:** No review-thread replies, resolutions, or comments were posted; only the branch code was updated after local verification.

## 2026-07-27 - PR #2 Merge and Workspace Sync

- **Task:** PR-2-MERGE-SYNC-001.
- **Skills:** `superpowers:receiving-code-review`, `superpowers:verification-before-completion`, `github:gh-address-comments`.
- **GitHub status:** PR #2 (`Config store budget`) is merged into `main` with merge commit `712f25732fda184272faa835d76343f568a3fd07`. The PR head was `c43215de93d635678b7691d2de886e14da1e801e`.
- **Review status:** The five original CodeRabbit inline threads were resolved by CodeRabbit after commit `c43215d`. The follow-up CodeRabbit run for the last seven changed files was not performed because the PR review limit was reached; the bot reported the next review window would reopen later rather than posting new actionable findings.
- **Workspace sync:** Fetched `origin/main` and `origin/config-store-budget`, fast-forwarded local `main` to `origin/main`, and fast-forwarded the `F:\codes\ai4se-cold-start` `config-store-budget` worktree from `b391b92` to `c43215d`.
- **Next step:** Task 5 should start from updated `main`, not from the already-merged `config-store-budget` branch.

## 2026-07-27 - Task 5: Policy Engine, Permission Profiles, and Exact Approvals

- **Task:** Task 5 implementation only.
- **Skills:** `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added `internal/policy/policy_test.go` before production files. With workspace-local Go caches, `go test ./internal/policy -v` failed as expected because `internal/policy` had no non-test Go files. A first invocation was blocked before compilation by the host Go build-cache permissions and was rerun with the workspace cache.
- **Green evidence:** `go test ./internal/policy -v` passed the profile matrix, hard denials, guarded mutation, dirty-worktree, SHA-256 digest binding, one-use approval, and repository-override tests. The common pre-Task-16 gate `go test ./... -count=1` and `git diff --check` passed.
- **Implementation:** Added a closed course-delivery action registry; risk-first hard denials for unknown actions, escape, `.git`, credential, binary, and unsafe facts; guarded normal limits and protected paths; review/supervised/workspace-auto mappings; deterministic sorted-baseline SHA-256 approval digests; and a mutex-protected, one-use `ApprovalStore` that also supports its zero value.
- **Self-review:** Confirmed policy has no repository configuration input that could loosen user-level decisions, approval grants bind run/profile/action/baselines, denied decisions contain no action arguments, and Task 6 workspace/tool execution was not started.
- **Concern:** The full Go gate initially timed out during dependency download to the workspace-local cache. It completed successfully after approved network/cache access; no dependencies were changed. The requested `git add`/`git commit` was blocked because the sandbox denied creation of `.git/index.lock`; the working tree remains cleanly modified for the controller to stage and commit.

## 2026-07-27 - Task 5 Independent Review and Closure

- **Task:** TASK-5-REVIEW-001.
- **Skills:** `superpowers:subagent-driven-development`, `superpowers:receiving-code-review`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Implementation commits:** `fa57720` added the initial policy engine and one-use approval store; `cf0f505` hardened `.git`, configured-check, key/endpoint-mismatch, and payload-derived limit handling; `730a9b2` required canonical mutation payloads; `933ce1c` rejected non-string `create_file` content.
- **Review loop:** The task reviewer found bypasses for dot-segment `.git` paths, arbitrary `run_check` command payloads, missing key/endpoint-mismatch hard denial, and mutation schema trust. Three scoped re-reviews verified each fix round; the final re-review returned all findings addressed with no new Critical or Important breakage.
- **Verification:** Controller-side focused policy tests, full `go test ./... -count=1` using workspace-local Go caches, and `git diff --check` passed after the final fix.
- **Human intervention:** None beyond approval for Git branch, staging, and commit operations blocked by the sandbox's `.git` write restrictions.
- **Lesson:** Policy code must validate canonical action payload shape itself; treating caller-supplied risk metadata as trusted creates the exact bypasses the policy layer is meant to prevent.

## 2026-07-27 - Task 6: Canonical Workspace and Read-Only Tools

- **Task:** Task 6 implementation.
- **Skills:** `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added workspace and tools tests before production code. With workspace-local Go caches, `go test ./internal/workspace ./internal/tools -v` failed as expected because both packages had no non-test Go files.
- **Implementation:** Added canonical workspace resolution with repository-escape, `.git`, protected-secret, and symlink-escape denial; read-only list/search/read tools with bounded deterministic results, UTF-8/binary checks, truncation, and SHA-256 content hashes; dispatch registry; Git baseline reads; and a temporary committed-repository test helper limited to test code.
- **Green evidence:** The focused workspace/tools suite passed after the minimum implementation. Full verification and commit status are recorded in the Task 6 report.
- **Self-review:** Traversal skips Git internals and symlink entries, search reads only enumerated canonical workspace files, and production tool registration exposes no Git write, raw-shell, network, credential, patch, or create action.

## 2026-07-27 - Task 6 Independent Review and Closure

- **Task:** TASK-6-REVIEW-001.
- **Skills:** `superpowers:subagent-driven-development`, `superpowers:receiving-code-review`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Implementation commits:** `39e024e` added workspace path resolution, read-only tools, registry, Git baseline reads, and the testrepo helper; `e6c3c47` bounded list traversal and made truncation UTF-8 safe; `1c6920a` preserved deterministic global ordering for bounded list results.
- **Review loop:** The task reviewer found two Important issues: output-only list bounds and byte-level UTF-8 truncation. The first scoped re-review verified those fixes and the Windows symlink-test improvement, then caught a new deterministic-ordering regression. The second scoped re-review returned all findings addressed with no new Critical or Important breakage.
- **Verification:** Controller-side focused `go test ./internal/workspace ./internal/tools -v`, full `go test ./... -count=1`, `gofmt -l internal`, and `git diff --check` passed after the final fix. Windows symlink and chmod traversal tests skipped only for host privilege/ACL limitations.
- **Human intervention:** None beyond approval for Git staging and commit operations blocked by sandbox `.git` write restrictions.
- **Lesson:** Bounded filesystem traversal still needs a clear ordering invariant; stopping early before proving the sorted prefix trades one resource bug for a correctness bug.

## 2026-07-27 - Task 7: Safe Patch and File-Creation Tools

- **Task:** Task 7 implementation.
- **Skills:** `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added patch/create tests before production code. The requested focused command first hit the host Go build-cache ACL; with workspace-local Go caches it then failed as expected on missing `NewPatchTool`, `PatchLimits`, `NewCreateTool`, and Task 7 sentinels.
- **Implementation:** Added bounded unified-diff header/hunk validation, workspace-resolved target checks, stale SHA-256 baseline checks before Git, all-or-nothing `git apply` with an atomicity hash guard, deterministic changed-path output and binary-diff digest. Added exclusive create/write/fsync/close behavior with cleanup on failure and no overwrite.
- **Green evidence:** `go test ./internal/tools -run 'TestPatch|TestCreate' -v` passed clean apply, conflict, stale baseline, injected atomicity breach, unsafe patch forms, limits, protected paths, and no-overwrite coverage. Full gate and commit status are recorded in the Task 7 report.
- **Self-review:** No executor or validation work was added; production mutation remains limited to the Task 7 tool implementations and uses no reset, clean, revert, shell-action, network, or credential access path.

## 2026-07-27 - Task 7 Independent Review and Closure

- **Task:** TASK-7-REVIEW-001.
- **Skills:** `superpowers:subagent-driven-development`, `superpowers:receiving-code-review`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Implementation commits:** `c4ffddd` added safe patch and create-file tools; `bbffdeb` fixed hunk parsing so header-like payload lines are not misread as file headers.
- **Review loop:** The task reviewer found one Important parser issue: valid hunk payload lines beginning `--- ` or `+++ ` could be rejected as structural headers. A scoped re-review verified the fix and found no new Critical or Important breakage.
- **Verification:** Controller-side `go test ./internal/tools -run 'TestPatch|TestCreate' -v`, full `go test ./... -count=1`, `gofmt -l internal`, and `git diff --check` passed after the final fix.
- **Human intervention:** None beyond approval for Git staging and commit operations blocked by sandbox `.git` write restrictions.
- **Lesson:** Unified diff parsers must honor hunk line counts before interpreting header-looking text; patch payload can contain strings that look like file headers.

## 2026-07-27 - PR #3 CodeRabbit Follow-up and Workspace Sync

- **Task:** PR-3-CODERABBIT-FOLLOWUP-001.
- **Skills:** `github:gh-address-comments`, `superpowers:receiving-code-review`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Review intake:** CodeRabbit reported five actionable unresolved PR #3 threads: mode-change patch headers were not rejected, a post-apply `git diff --binary` failure could return an unclassified error after mutation, `GitBaseline` could wait forever on Git because it lacked context, nested `.git` directories could be traversed, and `workspace.Resolve` did not re-check protected canonical targets after symlink resolution.
- **Red evidence:** Regression tests were added before production fixes. With approved Go build-cache access, `go test ./internal/tools ./internal/workspace` failed on the new mode-change rejection, diff-failure rollback/classification, nested `.git` traversal, and the new `GitBaseline(context.Context, root)` contract.
- **Implementation:** Commit `ca12a83` rejects Git mode-change patch headers, restores or classifies mutations when diff capture fails after apply, runs Git baseline commands through `exec.CommandContext`, blocks `.git` at any path depth, skips nested Git directories during list/search traversal, and rejects symlinks whose canonical targets resolve to protected paths inside the workspace.
- **Verification:** `go test ./internal/tools ./internal/workspace` passed after the fix, followed by full `go test ./...` with exit code 0. Before this documentation/workspace commit, `go test ./...` and `git diff --check` also exited 0.
- **GitHub writes:** Commit `ca12a83` was pushed to `origin/codex/policy-tools` for PR #3. This documentation/workspace follow-up records the pushed review closure in `PLAN.md` and this log.
- **Workspace status:** After the CodeRabbit fix push, `git status --short --branch` reported `## codex/policy-tools...origin/codex/policy-tools`; no untracked or modified files remained before this documentation synchronization.

## 2026-07-27 - Task 8: Cross-Platform Restricted Process Executor

- **Task:** Task 8 implementation in worktree `F:\codes\ai4se-executor-feedback` on branch `codex/executor-feedback`.
- **Skills:** `superpowers:using-superpowers`, `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added executor tests before production code. With workspace-local Go caches, `go test ./internal/executor -v` failed as expected because `internal/executor` had no non-test Go files. A first invocation in the original worktree was blocked by the host Go build-cache ACL before compilation.
- **Implementation:** Added the `executor.Executor` contract, `Local` process runner, testable `Mock`, secret-filtered explicit environment inheritance, independent bounded stdout/stderr draining, timeout/cancellation observation codes, Linux process-group cleanup, Windows Job Object cleanup, and no-shell test helper coverage.
- **Green evidence:** In `F:\codes\ai4se-executor-feedback`, `go test ./internal/executor -v` passed stream separation, exit code capture, output bounds, secret environment filtering, timeout cleanup, and context-cancellation cleanup on Windows.

## 2026-07-27 - Task 9: Ordered Validation Pipeline

- **Task:** Task 9 implementation in worktree `F:\codes\ai4se-executor-feedback` on branch `codex/executor-feedback`.
- **Skills:** `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added validation pipeline tests before production code. With workspace-local Go caches, `go test ./internal/validation -v` failed as expected because `internal/validation` had no non-test Go files.
- **Implementation:** Added stable `StageResult` and `Result` types plus a `Pipeline` that runs ordered stages through `executor.Executor`, treats pass as `EXIT` with exit code zero, stops on required failures, records optional failures without blocking later required stages, and supports required-only final validation reruns.
- **Green evidence:** In `F:\codes\ai4se-executor-feedback`, `go test ./internal/validation -v` passed ordered fail-fast, final required rerun, optional failure continuation, missing-exit failure, and context-cancellation coverage.

## 2026-07-27 - Task 10: Deterministic Feedback Pipeline

- **Task:** Task 10 implementation in worktree `F:\codes\ai4se-executor-feedback` on branch `codex/executor-feedback`.
- **Skills:** `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:systematic-debugging`, `superpowers:verification-before-completion`.
- **Red evidence:** Added feedback tests before production code. With workspace-local Go caches, `go test ./internal/feedback -v` failed as expected because `internal/feedback` had no non-test Go files.
- **Implementation:** Added ANSI/path/timing/address normalization, exact-before-pattern redaction, stable SHA-256 fingerprints, priority-ordered classification, configured regex classifiers, bounded summaries, and first/last evidence compression with truncation marking.
- **Debugging note:** The first green run caught a real classifier ordering bug: `unexpected` contained `expected`, so a compile error was misclassified as `TEST_FAILURE`. The generic validation order now checks compile/type/lint/build signals before test assertion wording.
- **Green evidence:** In `F:\codes\ai4se-executor-feedback`, `go test ./internal/feedback -v` passed fingerprint stability, secret redaction, summary/evidence/fingerprint secrecy, required category coverage, configured regex classification, and output compression tests.

## 2026-07-27 - PR #4 Early-Merge Remediation and Tasks 8-10 Closure

- **Task:** PR-4-REMEDIATION-001. PR #3 was already merged at `b4cab7c`. PR #4 was merged at `085933e` while three CodeRabbit threads were still unresolved; duplicate Draft PR #5 was left open and unchanged. Work continued on `codex/executor-feedback-review-fixes` from that exact `origin/main` baseline, and Task 11 was not started.
- **Recovery safety:** The original dirty worktree was inspected in full and preserved before branch work. Its recoverable patch is `C:\Users\12155\AppData\Local\Temp\ai4se-pr4-remediation\working-tree-20260727-194612.patch` with SHA-256 `7A8089423F626BE97BB979C598A51093FA548800176AB00B7A05FB820A649FAD`; no reset, checkout, clean, or silent overwrite discarded the user's changes.
- **Inherited review decisions:** The output-truncation Major was valid and now propagates `boundedBuffer` truncation through `domain.Observation` into `StructuredFeedback`. The unsupported-platform Major was valid, but the suggested generic Unix support was narrowed to the documented Windows/Linux delivery scope: Darwin and FreeBSD now fail before process start with a stable unsupported-platform error. The UTF-8 Minor was valid and summary truncation now stays within the byte limit at a valid UTF-8 boundary, including invalid short input and non-positive limits.
- **Classifier scope:** Cached configured regular expressions and secret-safe invalid-rule diagnostics were retained because they were supported by the original PR #4 review submission and failing regression coverage. They were isolated in commit `2ee1114` rather than folded into the three inherited inline fixes.
- **Recovered RED evidence:** A detached temporary worktree at merge base `085933e` received regression tests only. Focused tests failed on lost executor truncation state, no-op unsupported-platform behavior, invalid UTF-8 truncation, and missing classifier diagnostics. The same tests passed on the remediation branch. Later CodeRabbit follow-up produced a second real RED: Ubuntu race run `30272866647` lost stderr in `TestExecutorCapturesSeparateStreamsAndExitCode`. Deterministic inherited-pipe tests then reproduced `exec.ErrWaitDelay`, unbounded drain, and `afterStart` cleanup blocking before the final implementation.
- **Implementation:** Commits `01c5707` through `d045e6e` repaired truncation propagation, explicit platform rejection, UTF-8 bounds, classifier diagnostics, Windows suspended Job Object assignment, Linux process-group cleanup, bounded diagnostics, cleanup error visibility, and JSON-escaped secret redaction. Commit `09c9177` replaced `StdoutPipe`/`StderrPipe` ownership with explicit `os.Pipe` lifecycle management: wait for the root process, close the process tree, then drain with context/timeout bounds; forced drain closure is marked truncated and becomes a terminal cleanup error when it is not an existing timeout/cancellation. Both checkout steps also disable credential persistence.
- **Platform choice:** Full macOS/BSD process-group support was rejected because it would expand the promised delivery target without native runtime evidence. Darwin/FreeBSD coverage is compile-time proof of explicit rejection only; Linux process-tree behavior is exercised natively in Ubuntu Actions, while Windows Job Object behavior is exercised locally and in Windows Actions.
- **Local GREEN evidence:** Fresh Windows verification passed: `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, and `git diff --check`. Linux amd64 with CGO disabled built all packages and compiled the executor tests; Linux, Darwin, and FreeBSD executor test binaries also compiled successfully.
- **CI evidence:** On `09c9177`, push run `30276696614` and pull-request run `30276709353` both passed Ubuntu, Windows, Darwin rejection, and FreeBSD rejection jobs. The earlier push-run Ubuntu race failure was rerun successfully before its root cause was fixed, and the final dual-event runs passed after the deterministic drain repair.
- **Review evidence:** The initial fresh specification review returned Approved and the separate code-quality/security review returned Ready with no Critical, Important, or Minor findings. CodeRabbit then raised two PR #6 findings: checkout credential persistence and premature pipe closure. The first was fixed directly; the second was validated against the Ubuntu failure but its suggested bare `wg.Wait()` reorder was rejected because inherited pipes could deadlock. Two scoped independent review rounds found and closed the direct-writer `ErrWaitDelay`, unbounded-drain, and `afterStart` failure-path regressions. Final verdicts were specification Approved and code quality Ready with no Critical or Important findings.
- **GitHub review state:** CodeRabbit automatically marked its checkout-credential thread addressed and resolved after `09c9177`. Its executor-drain thread remains unresolved because the requested incremental review was rate-limited for 14 minutes; the current code addresses the behavior and passed equivalent independent review, but no agent replied to or resolved that thread. Original PR #4 threads were only linked from Draft PR #6 and were not replied to or resolved.
- **Documentation and lesson:** Tasks 8-10 are marked complete only after local GREEN, dual-platform Actions, and independent reviews. The process deviation remains explicit: a green status or merged PR is not review closure, pipe ownership must be designed together with process-tree cleanup, and an external review suggestion must be tested for lifecycle regressions rather than applied mechanically.
