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

Create internal/delivery/gitlab_ci_test.go:

~~~go
package delivery_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type job struct {
	Image        string   `yaml:"image"`
	BeforeScript []string `yaml:"before_script"`
	Script       []string `yaml:"script"`
}

func TestGitLabCIContainsRequiredJobs(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	content, err := os.ReadFile(filepath.Join(root, ".gitlab-ci.yml"))
	if err != nil {
		t.Fatalf("read .gitlab-ci.yml: %v", err)
	}

	var config map[string]yaml.Node
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatalf("parse .gitlab-ci.yml: %v", err)
	}

	assertJob := func(name, image string, commands ...string) {
		t.Helper()
		node, exists := config[name]
		if !exists {
			t.Fatalf("missing %s job", name)
		}
		var got job
		if err := node.Decode(&got); err != nil {
			t.Fatalf("decode %s job: %v", name, err)
		}
		if got.Image != image {
			t.Fatalf("%s image = %q, want %q", name, got.Image, image)
		}
		commandsInJob := append(append([]string(nil), got.BeforeScript...), got.Script...)
		joined := strings.Join(commandsInJob, "\n")
		for _, command := range commands {
			if !strings.Contains(joined, command) {
				t.Errorf("%s script missing %q", name, command)
			}
		}
	}

	assertJob("unit-test", "golang:1.26.5-bookworm", "go test ./... -count=1", "go vet ./...")
	assertJob("frontend-test", "node:24-bookworm", "npm --prefix web ci", "npm --prefix web test -- --run", "npm --prefix web run build")
}
~~~

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
  GOCACHE: "$CI_PROJECT_DIR/.cache/go-build"
  GOMODCACHE: "$CI_PROJECT_DIR/.cache/go-mod"
  NPM_CONFIG_CACHE: "$CI_PROJECT_DIR/.cache/npm"

cache:
  key: "$CI_COMMIT_REF_SLUG"
  paths:
    - .cache/

unit-test:
  stage: test
  image: golang:1.26.5-bookworm
  script:
    - go test ./... -count=1
    - go vet ./...

frontend-test:
  stage: test
  image: node:24-bookworm
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
$body = ($text -split "?
" | Where-Object { $_ -notmatch '^#' -and $_ -notmatch '^>' }) -join ''
$clean = $body -replace '[sp{P}p{S}]', ''
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
powershell -NoProfile -ExecutionPolicy Bypass -File .scripts	est.ps1
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

- [ ] **Step 1: Update local main after the reviewed PR merge**

~~~powershell
git switch main
git pull --ff-only origin main
git log -1 --oneline
~~~

Expected: main contains the migration design, GitLab CI, CI contract test, reflection, and active plan.

- [ ] **Step 2: Prove NJU did not advance independently**

~~~powershell
git ls-remote nju refs/heads/main
git merge-base --is-ancestor cadc7de8a7457bd0f59c21b33019e282f74b47d5 HEAD
~~~

Expected: NJU still points to the imported baseline and the baseline is an ancestor of reviewed main. Stop if NJU moved unexpectedly.

- [ ] **Step 3: Push reviewed main and verify the pipeline**

~~~powershell
git push nju main:main
~~~

Expected: a fast-forward push succeeds. Open the NJU pipeline page and confirm the latest pipeline is passed and both jobs passed; record its URL.

### Task 5: Publish and verify the CLI Release

**Files:**
- Create remotely: annotated tag v1.0.0 and GitHub Release assets.
- Modify after Release: REFLECTION.md release-status sentence only.

**Interfaces:**
- Consumes: synchronized green main and .github/workflows/release.yml.
- Produces: Windows/Linux binaries, checksums.txt, GHCR demo image, a public Release URL, and truthful final reflection status.

- [ ] **Step 1: Create and push the annotated tag**

~~~powershell
git tag -a v1.0.0 -m "AI4SE final project v1.0.0"
git push origin v1.0.0
~~~

Expected: GitHub starts the Release workflow for v1.0.0.

- [ ] **Step 2: Verify workflow and assets**

~~~powershell
gh run list --workflow release.yml --limit 5
gh release view v1.0.0 --json url,tagName,assets
~~~

Expected: workflow conclusion success; assets include ai4se-harness_windows_amd64.exe, ai4se-harness_linux_amd64, and checksums.txt.

- [ ] **Step 3: Download, verify, and smoke-test**

Download all three assets into a new temporary directory, compare the Windows binary SHA-256 with checksums.txt, then run:

~~~powershell
.ai4se-harness_windows_amd64.exe demo feedback-loop --format json
~~~

Expected: checksum matches and deterministic simulated feedback-loop JSON is returned without credentials.

- [ ] **Step 4: Make the reflection status truthful after Release**

Replace only the pending-release sentence in REFLECTION.md with the completed tag, checksum, and smoke-test evidence. Re-run the Task 2 reflection contract and keep the result within 1500–2500 characters.

- [ ] **Step 5: Integrate final documentation evidence**

Commit the reflection status update on a short review branch, merge through GitHub review, fast-forward NJU main, and verify the new NJU pipeline is passed. Record the final NJU repository, pipeline, and GitHub Release URLs together.
