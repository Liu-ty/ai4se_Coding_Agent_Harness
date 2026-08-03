# Submission Distribution and NJU GitLab Design

## Goal

Complete the remaining course-delivery path without changing the harness runtime: add the required NJU GitLab CI entry point, integrate `REFLECTION.md`, synchronize the reviewed result to the NJU repository, and publish a verifiable GitHub Release for the CLI distribution option allowed by the instructor's clarification.

## Repository Roles

- GitHub (`Liu-ty/ai4se_Coding_Agent_Harness`) remains the development and Release host. Its existing commit and pull-request history remains the authoritative development record.
- NJU GitLab (`noth1ng/ai4se_Coding_Agent_Harness`) is the course-submission repository. It must contain the same reviewed source commit, the required `.gitlab-ci.yml`, and a passing final pipeline.
- Local remotes stay explicit: `origin` identifies GitHub and `nju` identifies NJU GitLab. Neither remote is force-pushed.

## GitLab CI Design

Create a root `.gitlab-ci.yml` with a `test` stage and two independent jobs:

1. `unit-test` uses the repository's Go version and runs the deterministic Go unit suite plus `go vet`. The job name is exact because the course requires a job named `unit-test`.
2. `frontend-test` uses Node 24, installs the locked frontend dependencies, runs all Vitest tests, and builds the embedded WebUI assets.

Both jobs use repository-local caches only. They do not print environment variables, use real provider credentials, publish packages, deploy services, or mutate external state. GitHub Actions continues to provide the broader Windows, Linux, integration, Docker, secret-scan, build, and browser matrix.

## Change and Review Flow

Work occurs on `codex/submission-readiness`. The branch contains only the final `REFLECTION.md`, the migration design/plan, and `.gitlab-ci.yml`; local Go caches and unrelated files are excluded. Before any push, the full local verification script must pass. The branch is then reviewed through GitHub before the reviewed commit is synchronized to NJU GitLab.

## Release Flow

After the reviewed submission commit is present on `main`, create an annotated semantic version tag starting with `v`. The existing GitHub Release workflow builds Windows amd64 and Linux amd64 CLI binaries, generates SHA-256 checksums and release notes, and publishes the mock-only demo image to GHCR. A Release is complete only after the workflow succeeds and the public Release page exposes the expected artifacts.

The final course submission records:

- the NJU GitLab repository URL;
- the passing NJU pipeline URL;
- the GitHub Release URL allowed by the instructor's CLI-only clarification;
- target platforms, checksums, unsigned-binary limitation, and credential setup instructions already described by `README.md`.

## Verification and Failure Handling

- Verify the imported NJU `main` SHA equals the GitHub/local `main` SHA before making changes.
- Validate the CI contract locally: `.gitlab-ci.yml` exists, parses as YAML, declares `unit-test`, and includes the expected Go and frontend commands.
- Run `scripts/test.ps1` before integration.
- Push without force. If a remote has advanced, stop and reconcile rather than overwriting it.
- Treat a failed GitLab pipeline or GitHub Release workflow as incomplete delivery; inspect the job logs, fix on the branch, and repeat verification.
- Never enter, store, echo, or commit GitHub/GitLab tokens or provider API keys.

## Non-Goals

- No new harness behavior, provider, credential flow, or public real-execution capability.
- No production WebUI deployment; the CLI GitHub Release is the selected submission method under the instructor's clarification.
- No migration or rewriting of Git history, and no recreation of historical GitHub pull requests in GitLab.
