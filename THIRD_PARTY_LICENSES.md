# Third-Party Licenses and Provenance

Go dependency versions are locked in `go.mod`/`go.sum`; npm dependency versions are locked in `web/package-lock.json`.

| Component | Provenance | License |
|---|---|---|
| Go | golang.org | BSD-3-Clause |
| React, React DOM | npmjs.com | MIT |
| Vite, Vitest, TypeScript | npmjs.com | MIT |
| Playwright | npmjs.com | Apache-2.0 |
| modernc.org/sqlite | Go module proxy | BSD-3-Clause |
| BurntSushi/toml | Go module proxy | MIT |
| zalando/go-keyring | Go module proxy | MIT |
| golang.org/x/crypto, x/sys, x/term | Go module proxy | BSD-3-Clause |
| gopkg.in/yaml.v3 | Go module proxy | MIT and Apache-2.0 |
| Caddy | caddyserver.com | Apache-2.0 |

The UI design references Open Design's `dashboard` prototype and `linear-app` system only as design inspiration. Runtime assets and implementation are repository-owned; see `DESIGN.md`.
