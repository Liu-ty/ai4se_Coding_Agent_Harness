# AI4SE Coding Agent Harness — Codex Session Roadmap

**Date:** 2026-07-23
**Repository:** `git@github.com:Liu-ty/ai4se_Coding_Agent_Harness.git`
**Integration worktree:** `F:\codes\ai4se-cold-start`
**Branch lineage:** `cold-start/evidence-20260723` → `foundation` → `config-store-budget`
**Authoritative implementation plan:** `PLAN.md`

**Execution update:** The cold-start evidence and Task 1 foundation have been
split into local commits. Task 2 is isolated on `config-store-budget`; Task 3
remains blocked until the process metadata is closed and the foundation is
reviewed.

## Objective

Complete the course project through small, reviewable pull requests while preserving:

- the cold-start experiment and remediation evidence;
- strict red-green-refactor evidence for every implementation task;
- Windows x64 and Linux x64 compatibility;
- the course-required feedback loop, permission model, mock mechanism demo, WebUI, CI, release binaries, and container image;
- clean extension seams for a later full coding agent.

This roadmap coordinates sessions and Git integration. It does not replace the task definitions, interfaces, tests, or acceptance criteria in `PLAN.md`.

## Current Gate

The local split now has three auditable layers:

1. cold-start evidence and approved plan/spec revisions;
2. Task 1 foundation;
3. Task 2 configuration implementation on its dependent branch.

Before Task 3 begins, the remaining gate is to close the Task 1/2 metadata,
run independent verification, publish the protected branches, and review the
Foundation PR. The agent must not silently combine later PR groups merely to
simplify Git history.

## Session Boundary Rule

Use one primary Codex session for one PR/worktree objective. Start a fresh session when any of these is true:

- the target worktree or branch changes;
- the next numbered task belongs to another PR group;
- implementation is complete and an independent review is beginning;
- a merged interface changes the context needed by downstream tasks;
- the current session contains substantial discarded approaches or unrelated debugging history;
- the task requires an intentionally independent acceptance judgment.

Continue the existing session when:

- fixing review findings for the same task and branch;
- rerunning tests or formatting after a small local correction;
- answering a narrow question about the same active diff;
- completing documentation and `AGENT_LOG.md` for the same task.

Do not use “one session per day” or “one session per file.” The unit of continuity is one coherent objective on one worktree.

## Recommended Model Policy

The current Codex environment exposes GPT-5.6 Terra and GPT-5.6 Luna. The practical default is:

- **GPT-5.6 Terra, high:** normal implementation, TDD, and documentation.
- **GPT-5.6 Terra, xhigh:** Git history surgery, cross-platform process execution, policy/security, agent loop, credentials, API security, and difficult debugging.
- **GPT-5.6 Terra, max:** final independent acceptance, threat-focused review, or a stubborn failure after high/xhigh has produced insufficient evidence.
- **GPT-5.6 Luna, medium/high:** mechanical repository inspection, routine test reruns, formatting, simple documentation consistency, or low-risk scaffolding. Do not use it as the sole authority for security-critical or architecture-critical acceptance.

If GPT-5.6 Sol becomes selectable in Codex, prefer it for the hardest architecture/security/final-review sessions. Until then, Terra xhigh/max is the available quality-first choice.

Do not select maximum effort automatically. Begin at high for normal implementation and increase only when the task’s risk or observed failures justify it.

## Execution Sequence

| Stage | Worktree / PR | Tasks | New primary session? | Suggested model |
|---|---|---:|---|---|
| 0 | cold-start integration | evidence, split, verification | Yes, now | Terra xhigh |
| 1 | `foundation` | 1 | Continue Stage 0 only while performing the approved split; independent review gets a new session | Terra high; review xhigh |
| 2A | `config-store-budget` | 2–4 | Yes after Task 1 baseline | Terra high |
| 2B | `policy-tools` | 5–7 | Yes after Task 1 baseline | Terra xhigh |
| 2C | `executor-feedback` | 8–10 | Yes after Task 1 baseline | Terra xhigh |
| 3 | integration gate | verify Tasks 2–10 together | Yes, read/review focused | Terra max |
| 4 | `agent-providers` | 11–12 | Yes after Tasks 2–10 merge | Terra xhigh |
| 5 | `credentials-app` | 13–14 | Yes after Task 11 contracts freeze | Terra xhigh |
| 6 | `api-ui` | 15–16 | Yes after Tasks 11–14 merge | Terra high; API security review xhigh |
| 7 | `demo-release` | 17–18 | Yes after Tasks 15–16 merge | Terra xhigh |
| 8 | final course acceptance | all requirements | Yes, independent review | Terra max |

Stages 2A, 2B, and 2C may proceed in parallel only in separate worktrees after the Task 1 foundation is fixed. Avoid multiple write-capable agents in one worktree. Task 12 and Task 13 may proceed in parallel only after Task 11 freezes the provider and credential contracts.

## Per-Task Operating Loop

For every numbered task:

1. Read `SPEC.md`, the global constraints and exact task section in `PLAN.md`, and the latest relevant `AGENT_LOG.md` entries.
2. Inspect current branch, worktree status, and recent commits.
3. State the task boundary and list expected files before editing.
4. Write or activate the smallest failing test.
5. Run the targeted test and record the real red result.
6. Implement the minimum behavior needed for green.
7. Run the targeted test, then the common exit gate from `PLAN.md`.
8. Perform spec-compliance review.
9. Perform code-quality/security/cross-platform review.
10. Fix critical findings and rerun verification.
11. Update the task checkbox/hash and append an honest `AGENT_LOG.md` entry.
12. Commit only explicit task files. Push or open a PR only when the user has authorized it.

Never invent red/green evidence. Never modify the student-only `REFLECTION.md`.

## Prompt A — Cold-Start Evidence Integration

```text
你正在处理 AI4SE Coding Agent Harness 的冷启动证据整合，不是在开始新功能。

工作目录：F:\codes\ai4se-cold-start
当前分支：cold-start/opencode-clean-20260722-rerun
基线提交：dee5d29

请先使用 superpowers:using-superpowers，并按需使用
superpowers:using-git-worktrees、superpowers:finishing-a-development-branch
和 superpowers:verification-before-completion。先检查技能说明，再采取行动。

目标：
1. 审查当前所有未提交变更，保护现有成果，不得 reset、checkout 丢弃、clean 或删除未知文件。
2. 对照 SPEC.md、PLAN.md、SPEC_PROCESS.md、COLD_START_REPORT.md、AGENT_LOG.md，确认冷启动实验、独立修复、Task 1、Task 2 的归属。
3. 设计并执行可审计的拆分，使：
   - 冷启动证据与确认过的文档修订先形成独立提交；
   - Task 1 进入 foundation 分支/PR 谱系；
   - Task 2 进入 config-store-budget 分支/PR 谱系。
4. 特别处理 go.mod、go.sum、AGENT_LOG.md、PLAN.md 中跨任务混合的内容；不得为了省事静默合并 PR 范围。
5. 每次拆分后运行与范围相称的测试；最终至少运行：
   go test ./... -count=1
   go test -race ./... -count=1
   go vet ./...
   go mod verify
   git diff --check
   并验证 Linux amd64 交叉编译。

约束：
- 不开始 Task 3 或任何后续功能。
- 不改写 main 历史，不强推。
- 不推送、不创建 PR，除非我在本会话中明确授权。
- 若安全拆分存在冲突，先列出具体文件/hunk、风险和建议顺序，再等待确认；不要猜。
- 所有结论必须引用实际命令输出或 diff。

完成条件：
- 冷启动证据已经形成可追溯提交；
- foundation 与 config-store-budget 的边界清楚；
- Task 1/2 的代码和文档没有丢失；
- 验证通过；
- 最终报告列出分支、提交、文件归属、测试结果、尚未执行的 push/PR。
```

## Prompt B — Numbered Task Implementation

Replace the bracketed fields before use.

```text
请在 AI4SE Coding Agent Harness 中完成 PLAN.md 的 Task [编号：标题]。

工作目录：[绝对工作树路径]
目标分支：[分支名]
允许范围：[本 Task 的文件与行为]

请使用 superpowers:using-superpowers、superpowers:executing-plans、
superpowers:test-driven-development 和
superpowers:verification-before-completion；完成实现后使用
superpowers:requesting-code-review。若任务可拆为互不冲突的独立工作流，
仅在不同工作树/文件边界明确时才使用并行代理。

开始前：
1. 阅读仓库根目录 SPEC.md、PLAN.md 的 Global Constraints、Common Task Exit Gate、Task [编号] 完整章节，以及 AGENT_LOG.md。
2. 检查当前分支、工作树状态和最近提交。保留所有不属于你的现有改动；如范围重叠，先报告。
3. 用一段话复述任务目标、依赖、接口、测试和禁止事项。若 PLAN.md 已明确，不要反复向我询问。

执行要求：
- 严格 red-green-refactor：先写最小失败测试并实际运行，记录真实失败；再写最小实现；不得补写或虚构 red 证据。
- 只完成 Task [编号]，不要顺手开始下一 Task。
- 保持语言无关的产品行为，并同时考虑 Windows x64 与 Linux x64。
- 遵守安全边界：无任意 shell、无仓库逃逸、无秘密泄露、无未授权网络/依赖安装/Git 写操作。
- 使用 PLAN.md 指定的精确接口、错误、顺序和验收条件。
- 完成后先做规格一致性审查，再做代码质量/错误路径/安全/跨平台审查，修复所有关键问题。
- 运行 Task 专项测试及 PLAN.md Common Task Exit Gate；报告实际输出。
- 更新 Task checkbox/hash 和 AGENT_LOG.md，但不得代写 REFLECTION.md。
- 只提交明确属于本 Task 的文件。未经我明确授权，不 push、不创建 PR。

完成条件：
- Task [编号] 的所有验收条件有测试或可复核证据；
- 全部要求的检查通过；
- diff 中没有越界功能；
- 最终报告包含 red、green、全量验证、修改文件、提交哈希、风险与后续依赖。

如发生会改变公开接口、安全模型、PR 边界或课程范围的歧义，请停止实现并只提出一个具体问题；其他低风险细节按现有规范作合理决定。
```

## Prompt C — Independent Review and Acceptance

```text
请对 AI4SE Coding Agent Harness 的当前 [分支/PR/提交范围] 做独立验收。你是审查者，不是原实现者。

工作目录：[绝对工作树路径]
比较基线：[base 分支或提交]
审查范围：[Task 编号或 PR 名称]

请使用 superpowers:using-superpowers、superpowers:requesting-code-review
和 superpowers:verification-before-completion。第一轮保持只读。

先阅读 SPEC.md、PLAN.md 对应任务、SPEC_PROCESS.md 和 AGENT_LOG.md，再检查完整 diff、测试与提交历史。

按以下顺序工作：
1. 规格一致性：逐条核对接口、状态转换、错误语义、验收测试、课程要求和范围边界。
2. 安全与跨平台：重点检查权限绕过、路径/仓库逃逸、秘密泄露、命令执行边界、Windows x64/Linux x64 差异、公共 demo 能力裁剪。
3. 反馈闭环：确认失败被结构化、去敏、指纹化并能改变后续动作；不得只有重试外观。
4. 测试质量：检查是否存在只测 happy path、宿主机依赖、伪造 red/green 记录或未覆盖错误路径。
5. 工程质量：检查错误包装、资源释放、并发、确定性、命名、重复和可维护性。
6. 独立运行相关专项测试和 PLAN.md 的完整退出门禁。

输出：
- 先列 findings，按 Critical / Important / Minor 排序，每项给出文件和准确行号、影响、复现/证据及最小修复建议。
- 再给出每项验收条件的 PASS/FAIL/NOT PROVEN 表。
- 最后给出 ACCEPT、ACCEPT WITH FIXES 或 REJECT。
- 没有问题时明确写“未发现可操作问题”，但仍报告测试证据与剩余风险。

限制：
- 第一轮只审查，不修改文件。
- 不把原实现者的自述当作证据。
- 不 push、不合并、不关闭或批准 PR。
```

## User Checkpoints

The student should explicitly review or authorize at these points:

1. Approve the Stage 0 split before destructive Git history manipulation.
2. Review the cold-start evidence commit.
3. Review each PR’s scope and independent findings before merge.
4. Confirm provider behavior using mock credentials before any real API key is configured.
5. Manually verify Windows x64 and Linux x64 release artifacts.
6. Author `REFLECTION.md` personally.
7. Approve deployment of the mock-only public demo and verify that no real execution or credential route is exposed.

## Completion Signal

The next implementation stage may begin only when:

- the current gate’s commit/PR is reviewed;
- the target branch is based on the expected dependency commit;
- the worktree is clean or its pre-existing changes are explicitly accounted for;
- the session prompt names exactly one worktree/PR objective;
- tests from the previous gate still pass.
