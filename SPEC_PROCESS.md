# Specification Process

**Project:** AI4SE Coding Agent Harness  
**Design session:** 2026-07-10 to 2026-07-12  
**Primary method:** Superpowers `brainstorming`

## 1. Process Summary

The specification was produced before implementation. The repository was initially empty and not yet a Git repository. The process began by reading the course-wide requirements and the Coding Agent Harness-specific requirements together, then progressively fixed product scope, primary contribution, runtime boundary, technology stack, provider support, editing protocol, validation loop, governance, credentials, UI, distribution, and roadmap.

The student confirmed each major design section before it was written into `SPEC.md`.

## 2. Key Brainstorming Questions

The most useful questions were:

- Which harness dimension should be the primary contribution: feedback, governance, or memory?
- Should the agent be language-specific or accept declarative validation commands?
- Should the low-resource public server execute real code or only demonstrate the mechanism safely?
- Should code changes use patches, whole-file writes, or arbitrary shell?
- What task scope retains real utility while preserving objective completion criteria?
- Which permissions should be selectable by the user, and which boundaries must remain hard denials?
- How can the same core remain extensible without implementing a premature plugin platform?

These questions changed the project from a broad “build a coding agent” idea into a validation-driven repair harness with a measurable contribution.

## 3. Key Iterations

### Iteration 1 — Choosing the Main Contribution

**Assistant question:** choose feedback loop, governance, or memory as the deep dimension.  
**Student decision:** “反馈闭环。”

**Effect on the design:**

- Completion is defined by objective checks rather than model claims.
- Failure normalization, classification, fingerprinting, deduplication, compression, regression detection, and stopping became first-class modules.
- Governance and memory remain runnable minimums rather than competing primary contributions.

### Iteration 2 — Separating Local Execution from the Public Demo

The student disclosed a 2-core, 2-GB Ubuntu server and a domain. Three deployment shapes were compared: local execution plus safe public demo, full cloud execution, and CLI-first with a minimal page.

**Student decision:** use local real execution and a public safe demonstration.

**Effect on the design:**

- The public server never executes visitor code or stores real keys.
- Local mode owns the repository and real provider calls.
- Local and demo modes share domain logic but use distinct composition roots and route registries.
- The 2-GB server remains sufficient because it runs one Go process with mock/in-memory adapters.

### Iteration 3 — Technology Stack

Python/FastAPI, TypeScript/Node, and Go were evaluated across runtime cost, subprocess safety, WebUI ergonomics, packaging, LLM ecosystem, and cross-platform distribution.

**Student decision:** Go harness core plus React/TypeScript WebUI.

**AI suggestion adopted:** embed the static WebUI in the Go binary and use the same service locally and in the demo image.

**Reason:** the project’s hard problems are deterministic orchestration, policy, process control, and packaging rather than ML-library integration. Go provides the best overall fit for Windows/Linux native releases and the small Ubuntu server.

### Iteration 4 — Provider Compatibility Without Feature Explosion

The design compared a single OpenAI-compatible adapter, native OpenAI plus Anthropic, and full parity including streaming and native tools.

**Student decision:** implement OpenAI-compatible, Anthropic Messages, and mock providers through one canonical JSON decision protocol; omit token-level streaming and vendor-native tool parity.

**AI suggestion modified:** broad compatibility was retained, but only at the normalized decision layer. This prevents provider differences from taking over the feedback-loop project.

### Iteration 5 — Editing Mechanism

Structured patching, whole-file replacement, and arbitrary shell editing were compared against Aider, Claude Code, Codex, and OpenHands patterns.

**Student decision:** patch-first editing, constrained whole-file creation/replacement as a fallback, and only preconfigured validation commands.

**Effect on the design:**

- Diffs are compact and reviewable.
- Patch conflicts and stale baselines become structured feedback.
- Arbitrary shell and full sandboxing remain roadmap items.

### Iteration 6 — First-Version and Final Task Scope

The alternatives were defect repair only, defects plus small features, and arbitrary coding work.

**Student decision:** the first runnable version handles reproducible test-driven defects; the course-delivery version adds small features only when executable acceptance checks exist.

**AI suggestion rejected:** arbitrary coding tasks were excluded because completion would become model-judged and the project would lose its objective-feedback thesis.

### Iteration 7 — Extensibility

The student asked whether a simple first version and course-complete delivery could still evolve into a complete usable agent.

**Student decision:** use a stable core with provider, tool, executor, context, policy, credential, and store ports; retain an explicit post-course roadmap.

**Constraint added:** no dynamic plugin SDK, message bus, Kubernetes, vector database, or unused generalization is implemented during course delivery.

### Iteration 8 — Permission Profiles

The design moved from one fixed HITL policy to user-selectable profiles inspired by mainstream coding agents.

**Student decision:** `review`, `supervised`, and `workspace-auto`; real `full-access` waits for a future sandbox executor.

**Effect on the design:** the risk classifier remains constant while the profile maps risk to allow, approval, or deny. Hard boundaries remain non-overridable.

### Iteration 9 — Cross-Platform, Distribution, Credentials, and Validation

**Student decisions:**

- Support Windows x64 and Linux x64.
- Publish native GitHub Release binaries plus a GHCR Docker image.
- Prefer OS keyrings with an encrypted-vault fallback.
- Use a staged fail-fast validation pipeline.
- Use dual budgets: model decisions plus mutation/validation cycles.

These decisions fixed the project’s portability and security acceptance criteria rather than leaving them as deployment afterthoughts.

## 4. Accepted AI Suggestions

- Make feedback, not broad autonomy, the main contribution.
- Use local real execution and a mock-only public deployment.
- Use Go for the core and embed the React build.
- Normalize providers behind one non-streaming decision protocol.
- Use patch-first structured edits with stale-baseline detection.
- Automatically validate after every mutation and rerun the entire required pipeline before success.
- Treat permission profiles as mappings over one risk engine.
- Record typed append-only events and persist a run snapshot plus event transactionally.
- Keep Docker execution, arbitrary shell, plugins, memory, Git automation, and multi-agent behavior on an explicit roadmap.

## 5. Suggestions Rejected or Narrowed by the Student

- Full cloud code execution was rejected due to security, resource, and scope cost.
- A broad arbitrary-task agent was narrowed to objectively verifiable repairs and small changes.
- Full multi-provider native tool/streaming parity was narrowed to normalized non-streaming JSON decisions.
- Arbitrary shell editing was rejected for course delivery.
- Fixed HITL behavior was replaced with user-selectable profiles.
- A branded project name was deferred; the course-oriented name remains `AI4SE Coding Agent Harness`.

## 6. Brainstorming Reflection

### What Worked Well

- One-at-a-time questions forced decisions at the points that most affected architecture.
- Comparing concrete alternatives exposed hidden costs in public code execution, full provider parity, and arbitrary shell access.
- Repeatedly returning to the course’s “mechanism must be code and mock-testable” criterion prevented the design from becoming a prompt wrapper.
- Separating first runnable increment, course-delivery scope, and post-course roadmap made extensibility compatible with a disciplined submission.

### What Was Unsatisfactory

- The process was lengthy because many product, security, course-process, and deployment decisions were coupled.
- Several brand-name candidates were already occupied and naming did not materially improve the engineering design; the student correctly deferred branding.
- Official documentation tooling for OpenAI could not be installed in the current environment because the local `codex.exe` invocation was denied, so official OpenAI web documentation was used as fallback.
- The public demo requirement forces UI/deployment work that is adjacent to, rather than central to, the feedback-loop contribution.

## 7. Cold-Start Validation Protocol

The approved protocol was:

1. Use a different agent type in a new session.
2. Provide only the committed `SPEC.md` and `PLAN.md`.
3. Do not provide this conversation, memory, or verbal explanation.
4. Ask it to select and attempt one or two plan tasks.
5. Require it to stop and ask at ambiguity rather than guess.
6. Record questions, divergent interpretations, output gap, and exact SPEC/PLAN revisions in this file.

### 7.1 Execution and Independent Verification

On 2026-07-22, OpenCode with DeepSeek V4 Pro received only the committed `SPEC.md`, `PLAN.md`, and an operational experiment envelope. It worked in the isolated branch `cold-start/opencode-clean-20260722-rerun` at baseline `dee5d29`, attempted Tasks 1 and 2, did not start Task 3, and did not commit, push, merge, or change branches. Two fresh read-only Oracle sessions reviewed the resulting files.

The cold agent produced runnable Task 1 and Task 2 artifacts, but its self-reported `PASS` was not accepted without independent verification. Verification on 2026-07-22 and remediation on 2026-07-23 found:

- `Load` returned `*Config` although the plan fixed the public interface as `Load(io.Reader) (Config, error)`.
- Windows-path rejection depended on the host implementation of `filepath.IsAbs`; moving the test to a Windows-only file hid the Linux behavior instead of satisfying the cross-host requirement.
- The plan's `ResolveStage` example omitted the positive timeout required by its own validator.
- The plan did not explicitly require validation of `default_profile`.
- `CommandSpec` omitted the classifier rules that downstream feedback classification consumes.
- Review-fix tests were written with production changes before their red state was observed.
- `using-superpowers` was reported as preloaded rather than invoked through the skill tool.
- Reviews used explicit file snapshots rather than a complete Git diff because most generated files were still untracked.
- OMO created temporary `.omo/run-continuation` state, and the generated Go files had not been formatted.

The student approved a moderate acceptance standard: retain the useful implementation, do not repeat the entire cold start, waive the non-destructive process deviations above after disclosure, but require security, public-interface, formatting, and evidence corrections before accepting the gate. The verifier then:

1. Added a compile-time contract test, observed the expected pointer/value signature failure, and changed `Load` to return `Config`.
2. Added platform-neutral Windows drive, UNC, Windows root-relative, and POSIX absolute-path cases, observed the expected `\rooted` failure, and implemented host-independent rejection.
3. Removed the Windows-only test, formatted all Go files, and removed temporary `.omo` state.
4. Corrected the plan and experiment evidence and performed a fresh focused verification.

### 7.2 Exact Plan Revisions

| Confirmed gap | Before | After |
|---|---|---|
| Override example | `ValidationStage` example omitted `Timeout` | Example includes `Timeout: "1m"` |
| Permission profile | No explicit invalid-profile case | Red suite rejects values outside the three defined profiles |
| Absolute paths | Generic Windows/POSIX wording | Platform-neutral tests enumerate drive-rooted, root-relative, UNC, and POSIX forms on every host |
| Resolved classifiers | `CommandSpec` omitted `Classifiers` | `CommandSpec` carries `[]ClassifierRule`, and tests verify preservation |
| Validation text | Relied on host path interpretation | Requires host-independent absolute-path detection |

No `SPEC.md` product-scope change was necessary. The gate is accepted with verifier remediation, but its evidence and plan revisions must be reviewed and committed before Task 3 or any other later implementation task begins.

## 8. Written Specification Review

The approved conversational design was materialized as root-level `SPEC.md`, self-reviewed for placeholders, contradictions, scope, and ambiguous security choices, and committed as `137fe2f` (`docs: define coding agent harness specification`). The student then explicitly reviewed and approved the written specification on 2026-07-12 before plan generation began.

Self-review corrections included:

- Separating the read-only `review` state path from repair baseline validation.
- Fixing deployment to Caddy instead of leaving two reverse-proxy alternatives.
- Fixing vault algorithms to Argon2id and XChaCha20-Poly1305.
- Fixing the SQLite and keyring dependencies used for CGO-free cross-platform delivery.
- Binding the module path and Git remote to `github.com/Liu-ty/ai4se_Coding_Agent_Harness`.

## 9. Implementation Plan Generation

After written SPEC approval, `superpowers:writing-plans` decomposed delivery into 18 test-driven tasks across eight worktree/PR groups. The plan fixes exact file ownership, produced/consumed interfaces, red tests, expected failures, minimal implementation behavior, green commands, review gates, and commits. It also places the different-agent cold-start validation as a hard gate before Task 1 and reserves `REFLECTION.md` as a human-only deliverable. The student subsequently approved the implementation plan. The cold-start gate was executed and accepted with verifier remediation as recorded above; the evidence commit is the remaining prerequisite before later tasks begin.

## 10. Disk-Loss Recovery Record

On 2026-07-21, the local repository was missing after storage loss. Recovery used two independent sources:

1. The original course requirement files retained on the user's desktop/downloads.
2. A surviving local Git-object snapshot.

The snapshot contained an exact four-file pre-implementation tree. Its recovered blob IDs are:

- `AGENT_LOG.md`: `a8a1e9662e59a09955bc1459c2f391893af4508a`
- `PLAN.md`: `8a50fca602f5cce97668ea914ebf7df73abf201b`
- `SPEC.md`: `62db4a3f95d93e4d8b59e6c31720e8abd740ab34`
- `SPEC_PROCESS.md`: `5f84ea4b739072599f39230f017d3c11c1e9f2c6`

The original commit objects `137fe2f` and `7e9c1a8` were not present in the surviving object store and cannot be truthfully recreated with their original hashes. The exact document blobs were committed into a new repository history as `87482f1` (`docs: restore project planning baseline`) with the original GitHub remote restored.

Before resuming the workflow, a recovery self-review strengthened Task 2 of `PLAN.md`: every stated configuration validator now has an explicit red-test requirement, timeout parsing must reject non-positive durations, target-OS resolution must propagate validation errors, and two independent reviews are required before commit. A provider-shaped test token was also replaced by an unmistakable canary marker to avoid secret-scanner false positives. These clarifications do not change the approved product scope.
