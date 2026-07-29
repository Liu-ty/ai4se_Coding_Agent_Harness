# Agent Log

This log records the AI-assisted engineering process. It is append-only by convention. No real credential or unredacted secret may be recorded.

## 2026-07-28 - PR #8 Merge, CI, Review, and Closure

- **Scope:** Pre-Task-13 closure for merged PR #8 (`4cd2455`, branch `codex/agent-providers`).
- **Merge and CI:** PR #8 merged on 2026-07-28 after the Task 11/12 implementation and follow-up commits `d45e9cd`, `ff151af`, `6fb7536`, `410b556`, and `b6353f3`; its prior task log records the passing task-level suite and review remediation.
- **Review closure:** GitHub GraphQL reported one remaining unresolved, outdated CodeRabbit thread (`PRRT_kwDOTV9Iy86USI9x`) on `internal/provider/httpclient.go`. Verified on the merge commit that `newHTTPProvider` uses `url.ParseRequestURI`, rejects parse failures, empty scheme/host, and non-HTTP(S) schemes before construction, and retains a defensive request-time check. A fresh isolated-cache `go test ./internal/provider -count=1 -v` passed. Replied with this evidence at `discussion_r3662930115` and resolved the thread; all PR #8 review threads are now resolved.
- **Plan synchronization:** Tasks 11 and 12 are now marked complete in `PLAN.md`, with their implementation and follow-up commit hashes recorded. Task 13 remains the next planned implementation task; Task 15 is not started.

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
- **Red evidence:** `go test ./internal/store -v` initially failed with `no non-test Go files in internal/store`, after the contract test was added and before production store code existed. A restricted-sandbox attempt first failed only because the external Go build cache was inaccessible; the same command outside that restriction produced the expected red result.
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
- **Environment note:** Sandboxed Go 1.26.5 repeatedly printed `error acquiring upload token` for `%APPDATA%\go\telemetry\local\upload.token` even when tests/builds exited 0. `GOPATH`, `GOCACHE`, and `GOMODCACHE` were redirected to `.superpowers/sdd` so module downloads and checksum state stayed inside the workspace; the telemetry warning did not correspond to a test, vet, module, or build failure.
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
- **Workspace sync:** Fetched `origin/main` and `origin/config-store-budget`, fast-forwarded local `main` to `origin/main`, and fast-forwarded the isolated `config-store-budget` worktree from `b391b92` to `c43215d`.
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

## 2026-07-28 - Task 11: Provider-Neutral Agent Loop and Conditional Mock E2E

- **Task:** Task 11 implementation only.
- **Skills:** `superpowers:executing-plans`, `superpowers:using-git-worktrees`, `superpowers:test-driven-development`, `superpowers:systematic-debugging`, and `superpowers:verification-before-completion`.
- **Red evidence:** The first focused agent test failed because the `internal/provider` port did not exist. Approval-resumption and no-progress tests then failed before their production dependencies were added.
- **Implementation:** Added the normalized non-streaming provider port and concurrency-safe conditional mock; a project-owned one-action loop; bounded context assembly; typed boundary events; protocol-repair and budget handling; feedback-driven re-decision; final validation before success; cancellation persistence; no-progress detection; and one-use, exact pending-action approval recovery using the existing policy approval digest/store.
- **Security and scope:** Approval recovery keeps pending actions in the loop process only and reuses the existing exact digest / one-use approval boundary. It adds no credential persistence, API, UI, application service, raw-shell executor, or third-party agent loop.
- **Green evidence:** `go test ./internal/agent -v` passed conditional patch change, protocol exhaustion, approval one-use, final regression, cancellation, and no-progress coverage before task-level full verification.

## 2026-07-28 - Task 12: OpenAI-Compatible and Anthropic Provider Adapters

- **Task:** Task 12 implementation only.
- **Red evidence:** Local HTTP contract tests initially failed because `NewOpenAI` and `NewAnthropic` did not exist.
- **Implementation:** Added injected-client OpenAI-compatible and Anthropic Messages adapters over the Task 11 provider port. Requests use endpoint-bound credential lookup, the required vendor authentication headers, JSON-only responses, a 1 MiB response cap, bounded normalized errors, and cross-host redirect rejection. Credential byte slices are cleared after request construction and error values never contain request bodies or headers.
- **Green evidence:** `go test ./internal/provider -v` passed OpenAI/Anthropic endpoint and header contracts, malformed/oversized response rejection, and the credential non-leak regression; `go test ./... -count=1` passed before task-level review.

## 2026-07-28 - PR #8 CodeRabbit Follow-up

- **Review remediation:** Hardened provider endpoint validation, finite HTTP timeouts, redirect scheme checks, independently owned credential-key wiping, gateway `/v1` normalization, model/max-token options, and full retry-context request envelopes.

## 2026-07-27 - Task 8: Cross-Platform Restricted Process Executor

- **Task:** Task 8 implementation in the isolated worktree on branch `codex/executor-feedback`.
- **Skills:** `superpowers:using-superpowers`, `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added executor tests before production code. With workspace-local Go caches, `go test ./internal/executor -v` failed as expected because `internal/executor` had no non-test Go files. A first invocation in the original worktree was blocked by the host Go build-cache ACL before compilation.
- **Implementation:** Added the `executor.Executor` contract, `Local` process runner, testable `Mock`, secret-filtered explicit environment inheritance, independent bounded stdout/stderr draining, timeout/cancellation observation codes, Linux process-group cleanup, Windows Job Object cleanup, and no-shell test helper coverage.
- **Green evidence:** `go test ./internal/executor -v` passed stream separation, exit code capture, output bounds, secret environment filtering, timeout cleanup, and context-cancellation cleanup on Windows.

## 2026-07-27 - Task 9: Ordered Validation Pipeline

- **Task:** Task 9 implementation in the isolated worktree on branch `codex/executor-feedback`.
- **Skills:** `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:verification-before-completion`.
- **Red evidence:** Added validation pipeline tests before production code. With workspace-local Go caches, `go test ./internal/validation -v` failed as expected because `internal/validation` had no non-test Go files.
- **Implementation:** Added stable `StageResult` and `Result` types plus a `Pipeline` that runs ordered stages through `executor.Executor`, treats pass as `EXIT` with exit code zero, stops on required failures, records optional failures without blocking later required stages, and supports required-only final validation reruns.
- **Green evidence:** `go test ./internal/validation -v` passed ordered fail-fast, final required rerun, optional failure continuation, missing-exit failure, and context-cancellation coverage.

## 2026-07-27 - Task 10: Deterministic Feedback Pipeline

- **Task:** Task 10 implementation in the isolated worktree on branch `codex/executor-feedback`.
- **Skills:** `superpowers:executing-plans`, `superpowers:test-driven-development`, `superpowers:systematic-debugging`, `superpowers:verification-before-completion`.
- **Red evidence:** Added feedback tests before production code. With workspace-local Go caches, `go test ./internal/feedback -v` failed as expected because `internal/feedback` had no non-test Go files.
- **Implementation:** Added ANSI/path/timing/address normalization, exact-before-pattern redaction, stable SHA-256 fingerprints, priority-ordered classification, configured regex classifiers, bounded summaries, and first/last evidence compression with truncation marking.
- **Debugging note:** The first green run caught a real classifier ordering bug: `unexpected` contained `expected`, so a compile error was misclassified as `TEST_FAILURE`. The generic validation order now checks compile/type/lint/build signals before test assertion wording.
- **Green evidence:** `go test ./internal/feedback -v` passed fingerprint stability, secret redaction, summary/evidence/fingerprint secrecy, required category coverage, configured regex classification, and output compression tests.

## 2026-07-27 - PR #4 Early-Merge Remediation and Tasks 8-10 Closure

- **Task:** PR-4-REMEDIATION-001. PR #3 was already merged at `b4cab7c`. PR #4 was merged at `085933e` while three CodeRabbit threads were still unresolved; duplicate Draft PR #5 was left open and unchanged. Work continued on `codex/executor-feedback-review-fixes` from that exact `origin/main` baseline, and Task 11 was not started.
- **Recovery safety:** The original dirty worktree was inspected in full and preserved before branch work. Its recoverable patch is `%LOCALAPPDATA%\Temp\ai4se-pr4-remediation\working-tree-20260727-194612.patch` with SHA-256 `7A8089423F626BE97BB979C598A51093FA548800176AB00B7A05FB820A649FAD`; no reset, checkout, clean, or silent overwrite discarded the user's changes.
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

## 2026-07-27 - PR #6 Final CI Timing Stabilization

- **Failure evidence:** On documentation HEAD `561a514`, push run `30277145301` failed only in the Windows race step because `TestExecutorExternalDeadlineIsTimeout` required a heartbeat file after a fixed two-second deadline, but the hosted runner terminated the process tree before the child received enough scheduling time for its first write. The same SHA's pull-request Windows job in run `30277155728` passed, and the failing job's ordinary suite passed before the race-only failure.
- **Root cause and scope:** Production timeout classification and cleanup had no failure evidence. The test coupled two independent facts—external `DeadlineExceeded` handling and the child's first scheduled write—inside the same two-second window, then treated cleanup-before-first-write as a product failure.
- **Approved fix:** Commit `5ffc0d5` changes only `TestExecutorExternalDeadlineIsTimeout`. It runs the real executor asynchronously under a four-second external deadline, first observes the real child heartbeat, then asserts `CodeTimeout`, bounded return, and stopped heartbeat. No mock, retry, production change, Task 11 work, or platform expansion was introduced.
- **Verification and reviews:** The focused race test passed three consecutive runs. Fresh full local tests, full race tests, vet, module verification, formatting, diff checks, Linux amd64 build, and Linux/Darwin/FreeBSD executor compilation all passed. Scoped independent specification and code-quality reviews returned Approved and Ready with no Critical, Important, or Minor findings.
- **Final CI:** On `5ffc0d5`, push run `30278199413` and pull-request run `30278206875` both passed Ubuntu, Windows, Darwin rejection, and FreeBSD rejection jobs. CodeRabbit skipped this test-only incremental review because the PR remains Draft; the scoped independent reviews provide the equivalent review evidence required by the remediation gate.

## 2026-07-27 - PR #6 CodeRabbit Incremental Review Follow-up

- **Review scope:** CodeRabbit run `360e833b-5fc8-4ced-a78d-fbd5bcb1c4ef` reviewed commits through `8f79ba2` and reported three actionable inline findings: a developer-specific path in this log, a discarded process-tree cleanup error after `afterStart` failure, and unchecked deferred pipe-close results. The earlier output-drain thread remained open but its proposed bare `wg.Wait()` reorder was still rejected because inherited writers can hold the pipes indefinitely; the explicit-pipe, root-wait, tree-close, bounded-drain sequence already addresses that behavior.
- **RED evidence:** `TestLocalAfterStartFailureDoesNotWaitForInheritedPipes` was extended first to require both `ErrProcessCleanup` and the injected cleanup cause. The focused test failed with only `assign process tree failed`, proving that the `controller.close()` error was discarded on this branch.
- **Implementation and GREEN:** The failure branch now joins an `ErrProcessCleanup`-wrapped close error with the original `afterStart` error before returning. Deferred pipe closes explicitly discard their return values for `errcheck`, and both developer-specific absolute paths found in the tracked log were replaced with environment-relative redactions. The focused test then passed.
- **Deferred nitpick:** The suggested shared asynchronous timeout-test helper was not introduced. It was labeled low-value, changes no behavior, and would couple tests whose setup and timing contracts intentionally differ; keeping this review fix minimal preserves clearer failure ownership.
- **Verification:** Fresh `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, secret-path search, Linux amd64 CGO-disabled build, and Linux/Darwin/FreeBSD executor test compilation all passed before commit.

## 2026-07-28 - PR #4/#5 Review Closure and Workspace Reconciliation

- **Task:** PR-4-PR-5-CLOSURE-001. The user explicitly requested resolution of PR #4 and PR #5 plus documentation and workspace synchronization. Task 11 was not started.
- **PR #6 merge:** After the final CodeRabbit review completed, all five PR #6 threads were resolved and the required Ubuntu/Windows checks were green. A fresh pre-merge `go test ./... -count=1` passed on head `790e38c`. PR #6 was merged through the standard merge method at `16e982cd3798217eaf728058f8401f2480162808`; the merge tree exactly matched the tested head tree.
- **PR #4 legacy threads:** All three inherited CodeRabbit threads received evidence-based replies and were resolved. The replies identify the merged remediation for executor output-truncation propagation (`discussion_r3662396557`), explicit rejection of unsupported non-Windows/Linux targets (`discussion_r3662396689`), and byte-bounded valid UTF-8 feedback truncation (`discussion_r3662396791`). The historical early merge at `085933e` remains recorded as a process deviation.
- **PR #5 disposition:** Duplicate Draft PR #5 had the same old implementation head as PR #4 and contained no unique changes requiring integration. It was commented and closed as superseded by PR #4 plus remediation PR #6. Its head branch was intentionally retained.
- **Recovery safety:** The original dirty-worktree backup remains at `%LOCALAPPDATA%\Temp\ai4se-pr4-remediation\working-tree-20260727-194612.patch` with SHA-256 `7A8089423F626BE97BB979C598A51093FA548800176AB00B7A05FB820A649FAD`. Workspace-local Go cache directories and unrelated branches were preserved; no reset, clean, prune, or branch deletion was performed.
- **Workspace synchronization:** After a no-prune fetch, local `main` was fast-forwarded from `712f257` to merged `origin/main` at `16e982c`. The primary checkout remains on `codex/policy-tools` so its untracked workspace-local Go caches are untouched. The follow-up documentation work uses the existing isolated worktree on `codex/pr4-pr5-closure`, based on the same merged commit.
- **Documentation:** `PLAN.md` now records the resolved PR #4/#6 review state, the PR #5 closure, the PR #6 merge, completed Tasks 8-10, and the unchanged Task 11 boundary.
- **Closure verification:** A fresh GitHub query reported PR #4 merged with 3 total/0 unresolved threads, PR #5 closed, and PR #6 merged with 5 total/0 unresolved threads plus successful CodeRabbit and CI status. The documentation branch passed `go test ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, and the developer-specific absolute-path scan.

## 2026-07-28 - Task 13: OS Keyring and Encrypted Vault Credentials

- **Task and worker:** Independent Task 13 implementer in the isolated credentials worktree; Task 14 and Task 15 remained out of scope.
- **Skills:** `superpowers:executing-plans`, `superpowers:using-git-worktrees`, `superpowers:test-driven-development`, `superpowers:systematic-debugging`, and `superpowers:verification-before-completion`.
- **Interface gate:** Task 12 already supplies provider ID and endpoint host through `CredentialSource.Get`, and Task 8 already strips secret-like child-process environment entries. No prior-task interface change was needed.
- **RED evidence:** Vault and secret-free status tests were written before production files. The required focused command failed first because `internal/credentials` had no non-test Go files, then the expanded contract failed on the missing credential types, constructors, and keyring classifier. Later targeted RED runs caught invalid Windows permission evidence, missing headless/locked Secret Service fallback classification, and swallowed unavailable status without a fallback.
- **Implementation:** Added normalized provider/host references, copy-safe memory storage, hidden add/update/status/clear/get service behavior, keyring-first selection, allowlist-only unavailable/locked fallback, invalid-key non-fallback handling, and an Argon2id/XChaCha20-Poly1305 vault with authenticated endpoint binding, bounded header parsing, best-effort key-buffer clearing, owner-only temporary files, sync, and atomic rename.
- **Windows security debugging:** Go mode bits cannot demonstrate Windows confidentiality. The final test inspects a protected DACL containing one owner ACE. Root-cause tracing also found that updating a protected DACL requires a handle with both `WRITE_DAC` and `READ_CONTROL`; the focused ACL test passed after that single permission correction.
- **GREEN evidence:** Fresh focused credential tests passed, the focused race suite passed, full `go test ./... -count=1` passed, and a Linux/amd64 credential test binary compiled. `git diff --check` and production-canary scans passed before commit.
- **Security boundary:** Status/JSON and stable error sentinels contain no credential. Vault tests prove plaintext credential and master-password material are absent from disk bytes. The credential package has no SQLite, logger/event, subprocess, environment, configuration, public-demo, or Git-write integration.
- **Review:** Specification and code-quality self-review found and fixed the Windows ACL, headless D-Bus classification, and unavailable-status issues. No critical or important finding remains. Keyring `UpdatedAt` is intentionally process-local because the OS adapter exposes no portable update timestamp.
- **Human intervention:** The parent confirmed that a Git commit cannot embed its own final SHA and directed the PLAN heading to use `complete (this commit)` so Task 13 remains one commit. The commit subject is `feat: store provider credentials securely`; the actual SHA is reported after creation.
- **Lesson:** Cross-platform “owner-only” storage must be tested through native ACL semantics, and fallback eligibility must use a narrow allowlist so malformed credentials never silently migrate to a weaker backend.

## 2026-07-28 - Task 13 Security Review Fix Round

- **Scope and review validation:** Reopened Task 13 only after a security review. Each of the nine reported issues was checked against the committed implementation before changes; all represented a real race, ownership, reconciliation, portability, or format weakness. Task 14 and Task 15 remained untouched.
- **RED evidence:** New focused tests first failed to build because the platform classifier and Windows replacement seams did not exist. After those seams were introduced, behavioral RED showed a memory read observing a cleared credential, two concurrent Adds both succeeding, an ambiguous primary Set silently writing fallback, split-brain Get returning stale primary data, Clear ignoring an unknown/duplicate copy, a reused password callback buffer causing decryption failure, one vault reference overwriting another, and weakened Argon2 parameters reaching decryption. A separate provider RED proved the credential-source result buffer remained live.
- **Ownership fixes:** Vault treats callback bytes as borrowed, immediately creates and later clears its own copy, and supports callback buffer reuse. `Service.Get` clones and clears the store result; the provider port documents caller ownership and the HTTP adapter clears both the source result and its request-scoped copy. The provider ownership regression changed from RED to GREEN.
- **Reconciliation and concurrency fixes:** Service operations now serialize by normalized reference. Add selects fallback only when primary unavailability was established before any write; an error from an attempted primary Set is returned without creating a second copy. Status/Get/Update reject simultaneous primary/fallback records as ambiguous. Clear attempts every configured or unknown backend and reports any copy it cannot prove deleted.
- **Storage and platform fixes:** Vault records use full SHA-256 ref-derived sibling paths, retain provider/host AAD, and can independently add/get/delete multiple refs. Windows replacement uses `ReplaceFileW` for existing files and a write-through non-replacing move for first creation; Unix retains same-directory rename. Windows errno and Linux Secret Service/D-Bus classifiers now live behind target build tags.
- **KDF and race fixes:** Vault v1 accepts only the exact Argon2id tuple (time 3, memory 64 MiB, threads 2, key length 32). MemoryStore clones while holding its read lock, preventing concurrent Set/Delete from clearing the source mid-copy.
- **GREEN evidence:** Focused credential tests passed including Windows ACL/replacement and all new regressions; the focused MemoryStore/Add race run passed; the provider ownership regression passed; Linux/amd64 credential tests compiled with Linux-only classifier coverage; and fresh `go test ./... -count=1` passed.
- **Commit policy:** These fixes amend the single Task 13 commit with subject `feat: store provider credentials securely`; PLAN remains `complete (this commit)` because a commit cannot contain its own final SHA.
- **Post-fix review minor:** The scoped security re-review found that per-reference service locks were never evicted. A new internal regression first observed 64 retained locks after 64 completed distinct-reference adds; `referenceLock.users` now removes the map entry when the final holder releases it. The focused regression and a focused race run passed before the final Task 13 amend.

## 2026-07-28 - Task 14: Application Service, Preflight, and Repository Locking

- **Task and worker:** Task 14 implementer worked in the existing isolated credentials worktree. Task 15 was not started. The parent explicitly authorized the prerequisite `Store.ListRuns` work already present in the worktree and the minimum store/domain/agent changes needed to make the application lifecycle atomic and real.
- **Skills and process:** Used the existing written plan with `superpowers:executing-plans`, strict `superpowers:test-driven-development`, `superpowers:systematic-debugging`, `superpowers:verification-before-completion`, and independent review through `superpowers:requesting-code-review`.
- **Initial RED evidence:** `go test ./internal/app -v` first failed because the app service, finding codes, repository locks, and loop controller did not exist. Subsequent regressions independently proved that status-only baselines ignored changed bytes, untracked content was ignored, `RunCreated` was absent, SQLite returned a nil empty run list, atomic create/event and conditional state updates were absent, dirty supervised runs did not pause for exact baseline approval, endpoint queries could reach persisted inputs, cancellation and approval were not serialized safely, moved repositories stranded injected locks, and local composition did not bind a real `agent.Loop`.
- **Prerequisite RED/GREEN:** Store contract tests drove atomic `CreateRunWithEvent`, conditional `UpdateRunIfState`, `ErrRunStateChanged`, and consistent non-nil `ListRuns` behavior in memory and SQLite. Domain tests added only the application lifecycle transitions required by supervised approval, cancellation, and recovery. Agent tests drove exact rejection semantics, durable pending-action retry, conditional state transitions, and `APPROVAL_REJECTED` feedback classification.
- **Implementation:** Added ordered, secret-safe preflight findings for canonical Git roots, bounded commit/worktree baselines, configuration and executable checks, endpoint-bound credential status, profiles, dirty-worktree policy, and data-directory writability. Untracked baseline capture uses one cumulative byte budget, canonical workspace containment, stable content hashing, and rejects escaping symlinks. Query-bearing or user-info endpoints are rejected before any run is persisted.
- **Lifecycle and locking:** `Service` atomically persists `RunCreated`, gates dirty supervised runs on an exact run/profile/repository/commit/diff digest, rechecks the baseline before starting, and retains repository ownership until a durable terminal state. Post-creation setup errors are CAS-stopped with a non-cancelled cleanup context before lock release. Startup recovery stops abandoned runs and releases locks by run identity even if the repository path moved.
- **Concrete local composition:** `NewLocal` validates and binds an `AgentLoopFactory` using `RunSetup`, a real `agent.Loop`, and its one-use `ApprovalStore`. Each session owns a persistent run context and activity completion signal across initial, resumed, and rejection-continuation execution. Cancellation signals first, waits for active work, and does not block unrelated runs. Recoverable digest/busy errors retain the session; fatal action or persistence errors stop the run and release ownership. Concurrent approval regression coverage proves a second request cannot delete or cancel the valid in-flight session.
- **Review remediation:** Independent Task 14 reviewer first reported one Critical and five Important lifecycle/security findings. RED tests then reproduced approval-time cancellation failure, cross-run blocking, lost sessions after a bad digest, stranded active runs after action/setup errors, premature rejection pending deletion, unbounded/symlink-following untracked capture, and unconditional agent state overwrite. After those fixes, the reviewer found one remaining Important mismatch for `ErrLoopSessionBusy`; a blocking double-approval RED reproduced it, and controller/service classification was aligned. The final review also prompted synchronous nil-factory validation.
- **GREEN evidence:** Focused app/agent tests and affected race suites passed after every remediation round. Fresh full `go test ./... -count=1` and `go test -race ./... -count=1` passed with native Windows ACL behavior and temporary `GOCACHE`/`GOMODCACHE`. The final commit gate also runs vet, module verification, recursive formatting, and `git diff --check`.
- **Commit policy:** Task 14 remains one commit with subject `feat: orchestrate safe local harness runs`; the PLAN heading uses `complete (this commit)` because a commit cannot contain its own final SHA.

## 2026-07-28 - Task 13/14 Final Review Remediation

- **Scope:** The user authorized remediation of the final security/lifecycle review while preserving the two implementation commits by amendment. Task 15 remains untouched. Vault/keyring durability and status fixes are folded into Task 13; provider transport, app preflight/locks, and agent approval-flow fixes are folded into Task 14.
- **RED/GREEN evidence:** New regressions first showed that arbitrary public HTTP provider endpoints accepted stored credentials, custom endpoints lacked acknowledgement, Keyring `Status` retrieved the secret account, vault directory durability hooks and committed-write semantics were absent, repository ownership was process-local, the real loop omitted preflight baselines from approval digests, and an approved mutation with failed validation became a fatal app error. Every focused ordinary and race suite for `internal/provider`, `internal/credentials`, `internal/app`, and `internal/agent` passed after the fixes.
- **Credential transport and vault:** Provider and app validation now permit credentials only over HTTPS or literal loopback-IP HTTP; user-info/query/fragment endpoints are rejected. Non-default endpoint hosts or ports require explicit confirmation. Keyring persists a non-secret presence marker so status never reads the credential account. Vault syncs the parent directory after replacement/deletion on Unix, and reports `ErrVaultCommitted` if post-replacement hardening fails after the new record is already durable.
- **Durable orchestration:** Repository locks are owner-only create-exclusive lease files under the user cache directory, with run-ID/key records, local coordination, and recovery that removes only a matching durable lease even if the original path disappeared. The composition controller binds captured commit/diff values into every real-loop approval digest. Approval-resumed validation failures now emit validation feedback, transition to deciding, and re-query the provider instead of forcing the app to stop the run.
- **Cross-platform evidence:** Linux/amd64 CGO-disabled test binaries for credential and app packages compiled after adding the Unix parent-directory fsync seam. Final full Windows ACL ordinary/race, vet, module, formatting, and diff gates are recorded with the amended commits.

## 2026-07-28 - PR #9 CI and CodeRabbit Remediation

- **Scope:** Rechecked every still-valid CI/CodeRabbit finding without replying to or resolving review threads. Task 15 remained untouched; credential fixes stay in Task 13 and application, policy, CI, and log fixes stay in Task 14.
- **RED/GREEN evidence:** New tests initially failed for an unrevokable approval grant, secret residue after a keyring presence-marker failure, missing parent-directory sync before a committed permission failure, and an empty real-loop check queue panic. Each focused regression passed after its minimal repair, together with the controller wrong-digest integration regression and native Windows DACL checks.
- **Security and durability:** Failed status-marker writes now remove the just-written keyring secret. Vault reads close their handle explicitly, sync the parent directory before returning a committed post-replacement permission error, and Windows tests verify a protected DACL with no inherited ACEs.
- **Orchestration and CI:** Nonterminal approval errors revoke their one-use grant; repository lease read/verify/remove/map cleanup is serialized by one mutex; empty local-loop checks report a failed validation result. CI now verifies that the Git checkout root matches the canonical workspace root. The helper-name compile finding was rechecked against the current package graph and was not reproducible.
- **Log hygiene:** Replaced developer-specific absolute worktree paths in this log with repository-relative or generic descriptions.
- **Hosted Windows follow-up:** The real-loop test compared the raw requested path to the intentionally canonical preflight root; it now canonicalizes the expected test value, accommodating long/8.3 path spelling differences without changing production path handling. Vault DACL construction now reads the file owner SID from its opened handle instead of assuming the process-token SID owns the file. The protected, single-ACE, no-inheritance assertion remains strict and verifies that ACE against the actual file owner.

## 2026-07-28 - PR #9 Post-Merge Thread Follow-up

- **Scope:** A thread-aware GraphQL audit after PR #9 merged found three unresolved, non-outdated CodeRabbit threads: one log-hygiene Minor and two credential/approval Important findings. This follow-up remains limited to Tasks 13-14; Task 15 was not started.
- **RED evidence:** `TestVaultStatusDoesNotRequestMasterPassword` first failed with `master password unavailable`, proving that a status-only call performed the full password/KDF/decryption path. `TestBusyApprovalDoesNotRevokeExistingSameDigestGrant` first failed because a busy same-digest request deleted a pre-existing legitimate grant.
- **Credential fix:** Vault status now uses the normalized record path's existence and modification time. It does not request or retain password/secret material; `Get` remains the authenticated decryption and provider/host binding boundary.
- **Approval fix:** Grant and nonterminal-error revocation now execute only inside a successfully admitted session invocation. A request rejected with `ErrLoopSessionBusy` never touches the active invocation's same-digest grant.
- **Documentation hygiene:** All developer-specific worker/worktree paths were replaced with generic descriptions, and `PLAN.md` records both post-merge Task 13-14 corrections.
- **Verification:** The two new RED tests turned GREEN; complete affected-package ordinary/race suites and fresh `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal`, `git diff --check`, and the absolute-path scan all passed before publication.

## 2026-07-29 - Task 15: Versioned HTTP API, SSE, and Local Web Security

- **Task and worker:** Sole Task 15 implementer in the isolated `codex/api-ui` worktree. Tasks 16 and 17 were not started, and no subagent was used.
- **Skills and baseline:** Used `superpowers:executing-plans`, `superpowers:using-git-worktrees`, `superpowers:test-driven-development`, and `superpowers:verification-before-completion`. Required project documents were read in full. The branch started exactly at `072ef3d`; the controller-owned Task 13/14 PLAN heading corrections to `9d4efca` and `f73a4df` were preserved.
- **Initial RED evidence:** `go test ./internal/httpapi -count=1 -v` failed with `no non-test Go files`, after behavior tests were added and before any HTTP production file existed.
- **Focused RED/GREEN cycles:** Tests first exposed decoded canary content in artifact responses, acceptance of `[::1]` despite the fixed `127.0.0.1` local bind, cross-origin reads, canary content in SSE event types, a credential submitted through the API reappearing in a later event, colliding injected bootstrap/session/CSRF tokens, and an overbroad demo constructor retaining local capabilities. Each regression failed for the named behavior before its minimum production fix and passed afterward.
- **Implementation:** Added every planned `/api/v1` run, cancellation, approval, event, artifact, configuration-validation, and credential route plus `/healthz`. Responses use stable lower-case JSON DTOs and a uniform bounded error envelope. Request bodies are strictly decoded under a configurable maximum. The persistence port and memory/SQLite adapters gained the minimum run-bound `GetArtifact` read seam required by the artifact route, with backend-neutral missing/context/copy behavior.
- **Local security:** `NewLocal` accepts only a literal `127.0.0.1` Host, creates independent 32-byte bootstrap/session/CSRF tokens, rejects token collisions, consumes bootstrap once, redirects to a clean URL, sets an HttpOnly SameSite=Strict cookie, validates Host and supplied Origin on reads, and requires matching Origin plus CSRF on mutations. CORS is never enabled.
- **SSE and demo safety:** SSE emits monotonic `id`, typed `event`, JSON `data`, heartbeat comments, exclusive `Last-Event-ID` replay, and exits on request cancellation/disconnect. Payloads and event types pass through the central redactor. Demo construction forcibly prunes credentials, arbitrary run creation, config validation, cancellation/approval, and artifacts, while allowing reads/events only for explicitly fixed run IDs.
- **Canary evidence:** Tests prove configured canaries are absent from JSON errors, credential status, decoded artifact content, SSE data, and SSE event types. A credential submitted through the mutation route is registered with the runtime redactor before later responses/events can expose it.
- **GREEN evidence:** `go test ./internal/httpapi ./internal/store ./internal/storeport -count=1 -v` passed; `go test -race ./internal/httpapi ./internal/store ./internal/storeport -count=1` passed; and fresh `go test ./... -count=1` passed across the repository.
- **Scope and generated assets:** No frontend, CLI, demo scenario, embedded `webdist`, generated asset, dependency, push, or PR work was performed. `internal/httpapi/webdist` remains ignored and uncommitted.

## 2026-07-29 - Task 16: Open Design React WebUI

- **Scope and design source:** Sole Task 16 implementer worked on `codex/api-ui`; Task 17 was not started and no subagent was used. Open Design project `ai4se-coding-agent-harness-dashboard` was queried explicitly, confirmed with the `linear-app` design system and generated plugin snapshot, and pulled in full from entry `ai4se-harness-dashboard.html`. Its generated nine-section design specification is materialized as root `DESIGN.md`; the binary preview and prototype HTML were not copied into the repository.
- **Open Design provenance:** Generation run `f16462a6-af7a-48df-9864-dda329e04f06` used agent `codex`, model `gpt-5.6-sol`, plugin `example-dashboard`, and design system `linear-app`; the pulled source snapshot was `0b11c01b-61c8-4caa-aed6-50eade46016c`.
- **Runtime and dependency discipline:** Every npm invocation was immediately preceded by bundled Node `v24.14.0`. React and React DOM are pinned to 19.2.0, Vite to 8.1.0, dependencies are project-local, and `package-lock.json` is committed. The initial install exposed that plugin-react 5.x rejects Vite 8; registry metadata selected compatible 6.0.4. The shipped production dependency audit reports zero vulnerabilities; development tooling retains the upstream audit findings disclosed by the installer rather than applying an unsafe forced upgrade.
- **RED evidence:** The first full Vitest run failed in all three suites because the client, page, and component modules did not exist. Focused TDD then exposed evidence rendering, test isolation, keyboard order, and long-lived SSE behavior. The SSE regression specifically timed out before response close because `response.text()` buffered the live stream; the streaming reader made that focused test green.
- **Implementation:** Added a typed frozen-contract fetch client and incremental SSE parser that reconnects with the latest sequence; dense fixed-light Dashboard, New Run, Run Detail, Credentials, and Demo Gallery pages; status icon plus text; bounded explicitly expanded redacted evidence; numeric budgets; a keyboard-focusable read-only diff; exact one-time approval actions; and visible SIMULATED boundaries. Capabilities, fixed demo IDs, and CSRF are injected runtime composition data because Task 15 intentionally defines no discovery route. Credential secrets use password fields and clear React state synchronously when submission begins.
- **Browser and accessibility evidence:** Chrome E2E covers the complete SIMULATED guardrail → failed validation → changed second action → success → final diff flow and a keyboard-only supervised draft/approval flow. Axe reports zero serious or critical violations on Dashboard, New Run, Credentials, Demo Gallery, and local/SIMULATED Run Detail contexts.
- **GREEN and size evidence:** Fresh Vitest, Vite production build, Playwright E2E, and full Go regression commands passed after the final SSE fix. Vite 8.1 reported 0.29 kB gzip HTML, 1.56 kB gzip CSS, and 64.60 kB gzip JavaScript (about 66.45 kB total), well below the 1.5 MiB compressed ceiling. Generated `internal/httpapi/webdist`, Playwright results, and TypeScript incremental files remain ignored and uncommitted.
- **Review:** Spec-compliance review found no frozen API changes and confirmed all Task 16 pages and safety requirements. Code-quality review found and repaired the live-SSE buffering defect; final review found no critical issue. Task 15 remains a separate commit, and `PLAN.md` records the original Task 16 implementation commit.

## 2026-07-29 - Task 16 review follow-up

- **Runtime wiring and operational states:** Replaced all non-demo fixture behavior with paginated run reads, selected-run snapshots, preflight-gated creation, mutation actions, resumable SSE lifecycle management, referenced read-only artifacts, credential status/add/update/clear operations, server timestamps, budgets, and terminal reasons. Dashboard, preflight, credential, and detail surfaces now expose bounded loading, delayed, failure, recovery, empty, populated, and truncation states. Demo mode is restricted to server-declared fixed runs, and every fixed event row is visibly marked `SIMULATED`.
- **Real-boundary verification:** The test-only Playwright server composes production Vite assets with the real Task 15 local and demo routers. Browser coverage proves unauthenticated rejection, one-time bootstrap/session establishment, Host/Origin/CSRF enforcement, paginated lists, preflight/create, SSE approval/rejection/cancellation, referenced artifacts, credential lifecycle, demo route pruning, fixed feedback events, and zero serious or critical axe findings. Client tests additionally prove immediate `AbortController`/reader cancellation, bounded connect/read/parse recovery, state transitions, SSE envelope timestamps, and latest-sequence replay.
- **Final GREEN and size evidence:** Every npm command reported bundled Node `v24.14.0`; fresh Vitest passed 22/22 tests, the Vite build passed at 69.98 kB total gzip, and real Go-composition Playwright passed 2/2 scenarios with axe. Fresh `go test ./... -count=1`, `go test -race ./internal/httpapi -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal web/e2e/server`, `git diff --check`, and the developer-specific absolute-path scan all passed.

## 2026-07-29 - Task 15 production approval and recent-runs follow-up

- **Scope:** Reopened Task 15 only for the unified-review findings. No frontend, Task 17, push, PR, or subagent work was performed.
- **RED evidence:** A production-loop test first failed because `ApprovalRequired` had no DTO and the lifecycle transition persisted `{}`. Cross-backend list tests first failed because `Store.ListRuns` had no query/page contract and returned a full ID-ordered slice. The first GREEN attempt additionally exposed that applying regex redaction to serialized JSON could invalidate the payload; redaction was moved to decoded JSON string values before canonical remarshal.
- **Approval contract:** The real governance loop now publishes a stable `ApprovalRequired` DTO containing the exact canonical digest, a canonical redacted action, sorted redacted affected files, risk plus a non-empty operator reason, and sorted baseline evidence bindings. The digest canonicalizes valid action JSON while binding the unredacted action and baseline evidence. Agent and `NewLocal` application tests parse these fields from durable events produced by the real loop, then approve using the published binding. Pending approval is installed before the durable awaiting state becomes observable and removed if that transition fails.
- **Recent-runs contract:** `Store.ListRuns` now accepts a backend-neutral query and returns a page. Memory and SQLite apply fixed-ID filtering before offset/limit, order by `updated_at DESC, id ASC`, and determine `has_more` using one extra row; SQLite performs filtering, `LIMIT`, and `OFFSET` in SQL. The HTTP collection route passes demo fixed IDs into the store query, preserving pagination semantics when hidden runs are interleaved. Startup recovery retains an explicit internal unbounded query. `PLAN.md` now records the collection route, bounds/defaults, exact envelope, ordering, demo filtering, and approval-event schema.
- **Verification:** Focused agent/store/app/httpapi/storeport and policy suites passed. Fresh `go test ./... -count=1` passed; `go test -race ./internal/agent ./internal/app ./internal/httpapi ./internal/policy ./internal/store ./internal/storeport -count=1` passed; `go vet ./...`, `go mod verify`, `gofmt -l internal web/e2e/server`, and `git diff --check` all passed.

## 2026-07-29 - Task 15 approval secret-boundary follow-up

- **Scope and root cause:** Reopened only Task 15 review issue I1. The approval builder instantiated `feedback.NewRedactor(nil)` locally, while runtime credential registration updated a Router-private replacement redactor; neither path reached the real app/agent loop. The DTO also persisted the complete canonical patch. No frontend, Task 17, push, PR, or subagent work was performed.
- **TDD evidence:** The first agent regression did not compile because `Dependencies` had no approval redactor injection seam; the first real-app regression did not compile because `NewLocal` had no central redactor parameter. A constructor RED then proved central injection was optional. The tests use the non-pattern canary `correct-horse-7429`, a patch larger than 12 KiB, durable event bytes, and a closed SQLite database/WAL scan.
- **Implementation:** `feedback.Redactor` is now a concurrency-safe shared runtime registry. HTTP accepts the same central instance and registers configured, session, CSRF, bootstrap, and newly submitted credential secrets into it. `app.NewLocal` requires that central instance and binds it into every real loop before execution. Approval actions persist only bounded display data: SHA-256, at most 2 KiB of redacted preview, and a truncation flag for patch/content; patch previews contain only hunk headers and never changed lines or a complete patch. Other text, affected files, and baseline evidence are independently bounded and redacted. The exact unredacted canonical action remains bound only by the approval digest and in-memory pending action.
- **Focused GREEN evidence:** The agent regression proved digest/action/files/risk/evidence remain usable, the preview is redacted and truncated, the event stays below 8 KiB, and no SQLite file contains the canary. The real `NewLocal` test proved factory-created loops inherit the central redactor, and the HTTP credential test proved runtime registration is visible through the shared instance. Complete affected feedback/agent/app/httpapi/store suites passed before final repository verification.
- **Final verification:** Fresh `go test ./... -count=1` passed; `go test -race ./internal/feedback ./internal/agent ./internal/app ./internal/httpapi ./internal/store -count=1` passed; `go vet ./...`, `go mod verify`, `gofmt -l internal web/e2e/server`, and `git diff --check` all passed.

## 2026-07-29 - Task 16 unified review remediation

- **Scope:** Reopened Task 16 only for the eight unified-review findings, preserving the later Task 15 production approval DTO and recent-runs contract. Task 17, push, PR creation, and subagents remained out of scope.
- **RED evidence:** Focused frontend regressions first failed for demo leakage, stale and out-of-order preflight/status responses, approval rejection handling, SSE-only snapshot changes, terminal reconnect exhaustion, and cursor-preserving manual reconnect. The first Go helper test failed to compile because real repository/application composition was absent.
- **Frontend corrections:** Demo Gallery now intersects the runtime-declared fixed IDs with server data and renders an explicit empty state. New Run and Credentials use abortable generation/fingerprint guards; only `CREDENTIAL_NOT_FOUND` maps to unconfigured, while other status failures remain retryable. Approval decisions await completion, disable both actions while pending, and recover inline after approve or reject failure.
- **SSE and selection safety:** Every accepted event refreshes the selected server snapshot. Selection and snapshot generations prevent an older run or artifact request from overwriting a newer selection. Exhausted connect, HTTP, or read retries publish a structured terminal failure, and manual reconnect preserves the last event sequence.
- **Real-boundary helper:** The Playwright helper now composes production `app.Service`, `agent.Loop`, store, credential service, and Task 15 HTTP routers around a temporary Git repository. Its deterministic mock provider produces a production `ApprovalRequired` event using the stable action, affected-files, risk, reason, baseline, and digest fields; approval resumes the real loop to success. Demo data remains a distinct fixed read-only composition.
- **Keyboard and accessibility evidence:** Playwright creates a supervised run through the real Go boundary using keyboard navigation, asserts form and approval focus order, approves once with Enter, observes the approval panel close and the server snapshot become succeeded, and verifies zero serious or critical axe findings. The separate demo scenario proves only fixed runs are marked `SIMULATED`.
- **Final verification:** Every npm command reported bundled Node `v24.14.0`. Fresh Vitest passed 35/35 tests, Vite production build passed at 70.89 kB total gzip, and Playwright passed 2/2 real-boundary scenarios. Fresh `go test ./... -count=1`, `go test -race ./internal/agent ./internal/app ./internal/httpapi ./web/e2e/server -count=1`, `go vet ./...`, `go mod verify`, `gofmt -l internal web/e2e/server`, `git diff --check`, and the developer-specific path scan all passed.
