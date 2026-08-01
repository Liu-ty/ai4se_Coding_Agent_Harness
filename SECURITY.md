# Security Policy

## Trust boundary

The local executor intentionally trusts the repository selected by the user and the configured validation commands. It cannot prevent a malicious test executable from reading local files or using the network. Do not use `workspace-auto` as a sandbox for untrusted repositories.

The public demo is a separate mock-only composition. It contains no credentials, local filesystem access, custom endpoints, repository uploads, or real subprocess executor.

## Reporting a vulnerability

Do not publish secrets or exploit details in a public issue. Submit a [private GitHub vulnerability report](https://github.com/Liu-ty/ai4se_Coding_Agent_Harness/security/advisories/new) with the affected version, reproduction steps, impact, and a safe contact address. The maintainer will acknowledge reports, coordinate a fix, and publish a release note after remediation.
