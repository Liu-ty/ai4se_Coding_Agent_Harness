# Submission Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the required NJU GitLab CI contract, integrate the final reflection, synchronize a reviewed submission commit, and publish a verified GitHub CLI Release.

**Architecture:** GitHub remains the development and Release host while NJU GitLab is the course-submission mirror. A test-only Go package parses .gitlab-ci.yml and fixes the required job contract deterministically; GitLab supplies the authoritative remote pipeline result. Release work starts only after the reviewed source is on both remotes and NJU CI is green.

**Tech Stack:** Git, GitHub Actions/CLI, NJU GitLab CI, Go 1.26.5, gopkg.in/yaml.v3, Node.js 24, npm/Vitest/Vite, PowerShell.

## Global Constraints

- Keep origin mapped to git@github.com:Liu-ty/ai4se_Coding_Agent_Harness.git and nju mapped to https://git.nju.edu.cn/noth1ng/ai4se_Coding_Agent_Harness.git.
- Never force-push, rewrite imported history, print tokens, or commit provider credentials.
- The GitLab job name must be exactly unit-test.
- The selected distribution path is native CLI binaries through a public GitHub Release; no production WebUI deployment is required under the instructor's clarification.
- Do not stage .codex-go-mod-cache/, .tmp-go-cache/, or the superseded reflection scaffold/plan documents.
- Stop on any failed local test, GitLab pipeline, or GitHub Release workflow.

---

### Task 1: Add a deterministic GitLab CI contract

**Files:**
- Create: internal/delivery/gitlab_ci_test.go
- Create: .gitlab-ci.yml
- Modify: .gitignore

**Interfaces:**
- Consumes: repository root layout, Go 1.26.5, Node 24, gopkg.in/yaml.v3.
- Produces: exact jobs unit-test and frontend-test; go test ./internal/delivery enforces their images and scripts.

- [x] **Step 1: Write the failing CI contract test**

Create `internal/delivery/gitlab_ci_test.go` so it parses the real YAML, checks
the two job names, images, isolated cache keys/paths, and required commands.
Compare trim-normalized `before_script` and `script` entries individually;
allow additional arguments only for `npm --prefix web ci`. Include a regression
test proving that `echo "go test ./... -count=1"` cannot satisfy the command
contract.

- [x] **Step 2: Run the focused test and verify RED**

~~~powershell
go test ./internal/delivery -run TestGitLabCIContainsRequiredJobs -v
~~~

Expected: FAIL with read .gitlab-ci.yml because the required file does not exist.

- [x] **Step 3: Add the minimal GitLab pipeline**

Create .gitlab-ci.yml:

~~~yaml
stages:
  - test

variables:
  GOCACHE: "$CI_PROJECT_DIR/.cache/go/build"
  GOMODCACHE: "$CI_PROJECT_DIR/.cache/go/mod"
  NPM_CONFIG_CACHE: "$CI_PROJECT_DIR/.cache/npm"

unit-test:
  stage: test
  image: golang:1.26.5-bookworm
  cache:
    key: "go-$CI_COMMIT_REF_SLUG"
    paths:
      - .cache/go/
  script:
    - go test ./... -count=1
    - go vet ./...

frontend-test:
  stage: test
  image: node:24-bookworm
  cache:
    key: "npm-$CI_COMMIT_REF_SLUG"
    paths:
      - .cache/npm/
  before_script:
    - npm --prefix web ci --prefer-offline
  script:
    - npm --prefix web test -- --run
    - npm --prefix web run build
~~~

Append .cache/ to .gitignore so a local CI-compatible run cannot pollute status.

- [x] **Step 4: Verify GREEN and the broader Go suite**

~~~powershell
go test ./internal/delivery -run TestGitLabCIContainsRequiredJobs -v
go test ./... -count=1
go vet ./...
~~~

Expected: all commands exit 0; the focused test reports PASS.

- [x] **Step 5: Commit the CI contract**

~~~powershell
git add -- .gitlab-ci.yml .gitignore internal/delivery/gitlab_ci_test.go
git diff --cached --check
git commit -m "ci: add NJU GitLab test pipeline" -m "Agent: Codex (no subagent). Human: approved the NJU migration and CLI Release delivery path."
~~~

### Task 2: Integrate the reviewed reflection and active plan

**Files:**
- Create: REFLECTION.md
- Create: docs/superpowers/plans/2026-08-03-submission-readiness.md

**Interfaces:**
- Consumes: the user-authored reflection draft and verified repository history.
- Produces: a 1500–2500-character reflection with eight sections and an explicit AI-polishing disclosure; an auditable active plan.

- [x] **Step 1: Run the reflection contract check**

~~~powershell
$text = Get-Content -LiteralPath REFLECTION.md -Raw -Encoding UTF8
$body = ($text -split "\r?\n" | Where-Object { $_ -notmatch '^#' -and $_ -notmatch '^>' }) -join ''
$clean = $body -replace '[\s\p{P}\p{S}]', ''
if ($clean.Length -lt 1500 -or $clean.Length -gt 2500) { throw "reflection length: $($clean.Length)" }
if ([regex]::Matches($text, '(?m)^## ').Count -ne 8) { throw 'reflection must have eight sections' }
if ($text -match '【请|TODO|TBD') { throw 'reflection contains placeholders' }
if ($text -notmatch 'AI 辅助披露') { throw 'reflection disclosure missing' }
~~~

Expected: exit 0 and no output.

- [x] **Step 2: Confirm the factual release boundary**

~~~powershell
rg -n "正式 tag|GitHub Release|尚待|仍须|AI 辅助披露" REFLECTION.md
git tag --list
gh release list --limit 10
~~~

Expected before release: the reflection says Release work is pending, while tag and Release listings are empty.

- [x] **Step 3: Commit only the final reflection and active plan**

~~~powershell
git add -- REFLECTION.md docs/superpowers/plans/2026-08-03-submission-readiness.md
git diff --cached --check
git commit -m "docs: add final project reflection" -m "Agent: Codex polished structure and checked repository facts; no subagent. Human: supplied and approved all first-person views."
~~~

### Task 3: Verify and publish the review branch

**Files:**
- Verify: all tracked project files.
- Exclude: local cache directories and superseded reflection scaffold documents listed in Global Constraints.

**Interfaces:**
- Consumes: Tasks 1–2 commits on codex/submission-readiness.
- Produces: green local verification and a GitHub draft PR targeting main.

- [x] **Step 1: Run the complete local verification**

~~~powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test.ps1
~~~

Expected: 51 frontend tests pass, all Go packages pass, go vet exits 0, and both Playwright tests pass.

- [x] **Step 2: Restore only generated embedded assets**

~~~powershell
git restore --worktree -- internal/httpapi/webdist/index.html internal/httpapi/webdist/assets/placeholder.js
git status --short --branch
~~~

Expected: no tracked source changes remain; only explicitly excluded untracked files may appear.

- [x] **Step 3: Push and open a draft PR**

~~~powershell
git push -u origin codex/submission-readiness
gh pr create --draft --base main --head codex/submission-readiness --title "ci: complete submission readiness" --body "## Summary`n- add the required NJU GitLab unit-test and frontend-test jobs`n- add a deterministic CI contract test`n- add the reviewed final reflection and delivery plan`n`n## Ownership`nCodex implemented and verified the CI/configuration work without subagents. The student supplied and approved all first-person reflection views.`n`n## Verification`n- scripts/test.ps1`n- reflection length, structure, placeholder, and disclosure contract"
~~~

The reviewed PR body must list the GitLab CI contract, reflection integration, AI/human ownership, and full verification evidence. Expected: a public draft PR URL.

### Task 4: Synchronize the reviewed commit and verify NJU CI

**Files:**
- Remote state only after the GitHub PR is reviewed and merged.

**Interfaces:**
- Consumes: merged GitHub main.
- Produces: the same commit on NJU main and a passing pipeline containing unit-test and frontend-test.

- [x] **Step 1: Update local main after the reviewed PR merge**

~~~powershell
git switch main
git pull --ff-only origin main
git log -1 --oneline
~~~

Expected: main contains the migration design, GitLab CI, CI contract test, reflection, and active plan.

- [x] **Step 2: Prove NJU did not advance independently**

~~~powershell
$expectedNjuMain = 'cadc7de8a7457bd0f59c21b33019e282f74b47d5'
$remoteLine = git ls-remote nju refs/heads/main
if ($LASTEXITCODE -ne 0 -or -not $remoteLine) { throw 'unable to read nju/main' }
$actualNjuMain = ($remoteLine -split '\s+')[0]
if ($actualNjuMain -ne $expectedNjuMain) { throw "nju/main moved: $actualNjuMain" }
git merge-base --is-ancestor $expectedNjuMain HEAD
if ($LASTEXITCODE -ne 0) { throw 'imported baseline is not an ancestor of HEAD' }
~~~

Expected: NJU still points to the imported baseline and the baseline is an ancestor of reviewed main. Stop if NJU moved unexpectedly.

- [x] **Step 3: Push reviewed main and verify the pipeline**

~~~powershell
git push nju main:main
~~~

Expected: a fast-forward push succeeds. Open the NJU pipeline page and confirm the latest pipeline is passed and both jobs passed; record its URL.

### Task 5: Publish and verify the CLI Release

**Files:**
- Create remotely: annotated tag v1.0.0 and GitHub Release assets.
- Create: .github/workflows/release-smoke.yml.
- Test: internal/delivery/release_smoke_workflow_test.go.
- Modify after Release: REFLECTION.md release-status sentence only.

**Interfaces:**
- Consumes: synchronized green main and .github/workflows/release.yml.
- Produces: Windows/Linux binaries, checksums.txt, GHCR demo image, a public Release URL, and truthful final reflection status.

- [x] **Step 1: Create and push the annotated tag**

~~~powershell
git tag -a v1.0.0 -m "AI4SE final project v1.0.0"
git push origin v1.0.0
~~~

Expected: GitHub starts the Release workflow for v1.0.0.

- [x] **Step 2: Verify workflow and assets**

~~~powershell
gh run list --workflow release.yml --limit 5
gh release view v1.0.0 --json url,tagName,assets
~~~

Expected: workflow conclusion success; assets include ai4se-harness_windows_amd64.exe, ai4se-harness_linux_amd64, and checksums.txt.

- [x] **Step 3: Download, verify, and smoke-test**

Download all three assets into a new temporary directory. Verify the SHA-256
listed in `checksums.txt` for both binaries. On Windows, compare each value with
`Get-FileHash -Algorithm SHA256`, then run:

~~~powershell
.\ai4se-harness_windows_amd64.exe demo feedback-loop --format json
~~~

Because the submission machine has no WSL distribution or Docker runtime, add
`.github/workflows/release-smoke.yml` with a required `tag` dispatch input and
an `ubuntu-latest` job. After merging that workflow, run:

~~~powershell
gh workflow run release-smoke.yml -f tag=v1.0.0
gh run list --workflow release-smoke.yml --limit 3
~~~

The runner downloads the same three assets and runs:

~~~bash
sha256sum --check dist/checksums.txt
chmod +x dist/ai4se-harness_linux_amd64
dist/ai4se-harness_linux_amd64 demo feedback-loop --format json
~~~

Expected: both checksums match, and both platform binaries return deterministic
simulated feedback-loop JSON without credentials. Release verification is not
complete until the Linux smoke test has run on Linux.

Evidence: the Windows binary and both downloaded SHA-256 values were verified
locally. GitHub Actions run
`https://github.com/Liu-ty/ai4se_Coding_Agent_Harness/actions/runs/31253948202`
then verified both assets as `OK` on Ubuntu 24.04 and returned terminal
`{"state":"SUCCEEDED"}` from the Linux binary.

- [x] **Step 4: Make the reflection status truthful after Release**

Replace only the pending-release sentence in REFLECTION.md with the completed tag, checksum, and smoke-test evidence. Re-run the Task 2 reflection contract and keep the result within 1500–2500 characters.

Evidence: Section 6 now records the completed `v1.0.0` tag, dual-platform
checksum verification, and Windows/Ubuntu smoke results without claiming a
production WebUI deployment.

- [x] **Step 5: Integrate final documentation evidence**

Commit the reflection status update on a short review branch, merge through GitHub review, fast-forward NJU main, and verify the new NJU pipeline is passed. Record the final NJU repository, pipeline, and GitHub Release URLs together.

Evidence: GitHub PR #16 merged the final reflection evidence at `bfcb9a6` after
all checks passed. NJU `main` fast-forwarded to the same commit, and pipeline
`https://git.nju.edu.cn/noth1ng/ai4se_Coding_Agent_Harness/-/pipelines/319070`
passed both `unit-test` and `frontend-test`. Final links:

- NJU repository: `https://git.nju.edu.cn/noth1ng/ai4se_Coding_Agent_Harness`
- GitHub Release: `https://github.com/Liu-ty/ai4se_Coding_Agent_Harness/releases/tag/v1.0.0`
