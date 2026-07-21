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
