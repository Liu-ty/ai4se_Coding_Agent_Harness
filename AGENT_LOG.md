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
