# AI4SE Coding Agent Harness

## Overview

AI4SE is a governed coding-agent harness. It applies only typed, policy-checked actions, turns objective check failures into structured feedback, and records an auditable run timeline.

## Why This Exists

The project demonstrates a deterministic feedback loop: a mock provider receives policy/validation feedback and changes its next patch. It favors narrow, verifiable mechanisms over an unrestricted autonomous shell.

## Architecture

The Go application service is shared by CLI and HTTP routes. Providers, executor, store, credentials, policy, validation, and feedback are injected at composition roots. `local` uses a restricted local repository runtime; `demo` is separately composed from an in-memory store, scripted provider, and in-memory patch executor.

## Installation

Download a matching GitHub Release binary and its SHA-256 checksum. Native users do not need Go or Node. Source development requires Go 1.26.5 and Node 24 LTS.

## Windows Quickstart

```powershell
.\ai4se-harness_windows_amd64.exe demo feedback-loop --format json
.\ai4se-harness_windows_amd64.exe serve --profile local --repo C:\path\to\repository
```

## Linux Quickstart

```bash
chmod +x ai4se-harness_linux_amd64
./ai4se-harness_linux_amd64 demo feedback-loop --format json
./ai4se-harness_linux_amd64 serve --profile local --repo /path/to/repository
```

## Configuration

Place a strict version-1 `.ai4se-harness.toml` in the repository. Define a default permission profile and ordered validation stages. Commands are argument arrays with platform-specific executable overrides; raw shell strings are never accepted.

## Validation Pipeline

Repair runs first reproduce a failing baseline, then execute configured validation after each mutation and the complete required pipeline before success. Failed observations are normalized, redacted, classified, fingerprinted, and fed into the next decision.

## Permission Profiles

`review` proposes without mutation; `supervised` requires approval; `workspace-auto` permits bounded workspace changes. Repository escape, credentials, arbitrary shell, network tools, binary writes, and protected paths remain hard-denied in every profile.

## Credential Security

Use `ai4se-harness credentials set|status|clear --provider <id> --endpoint <url>`. Input is hidden; keys are never flags, environment echoes, config values, events, child environments, logs, SQLite records, or demo data. Local storage prefers OS keyrings and falls back to an encrypted vault where supported.

## Running Locally

```text
ai4se-harness serve --profile local --repo <path>
ai4se-harness run --repo <path> --task <text> --config .ai4se-harness.toml
```

Local HTTP binds to loopback, requires a one-time bootstrap token, validates Host/Origin, and requires CSRF for mutations. A supervised CLI run that needs approval keeps its owning runtime alive and prints a temporary loopback continuation URL.

## Public Demo

```text
ai4se-harness demo feedback-loop --format text|json
ai4se-harness serve --profile demo --addr 0.0.0.0:8080
```

The public demo is always simulated. It registers no credential, filesystem, custom endpoint, repository-upload, or real process-execution capability; every workspace is in memory.

## Distribution

Tag `v*` to build `ai4se-harness_windows_amd64.exe`, `ai4se-harness_linux_amd64`, checksums, GitHub Release notes, and `ghcr.io/liu-ty/ai4se_coding_agent_harness:<tag>`.

## Directory Structure

- `cmd/ai4se-harness`: CLI and composition roots
- `internal`: domain, policy, feedback, persistence, API, local and demo adapters
- `web`: React/Vite UI
- `scripts`: reproducible developer verification
- `deploy`: Caddy and Compose deployment assets

## Security Boundaries

The local executor trusts the selected repository and its configured checks; it is not a sandbox for malicious tests. The demo never starts a real subprocess and is the only supported public deployment profile. See [SECURITY.md](SECURITY.md).

## Known Limitations

The delivery scope supports one local repository and configured checks, not arbitrary shell/network tools, containers for local execution, Git writes, PR automation, or unbounded project memory. A supervised approval must be completed while its CLI-owned local continuation server is running.

## Development

```bash
npm --prefix web ci
go test ./...
```

## Testing

```powershell
./scripts/test.ps1
```

```bash
./scripts/test.sh
```

Both commands run frontend unit tests, build embedded assets, Go tests, vet, and browser E2E tests in that order without printing environment variables.

## CI/CD

GitHub Actions runs `unit-test` on Windows and Ubuntu, frontend, secret, integration, build, Docker, and browser jobs. CI uses Node 24 and rebuilds embedded web assets before Go compilation.

## Deployment

Set `AI4SE_DOMAIN` and `AI4SE_IMAGE_TAG`, then run `docker compose -f deploy/compose.yml up -d`. Caddy obtains TLS for the exact domain and proxies only to the mock-only demo. The demo container uses a read-only root filesystem, tmpfs, no capabilities, no-new-privileges, and CPU/memory/PID limits. Retain the previous image tag for rollback.

## Third-Party Licenses

See [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).
