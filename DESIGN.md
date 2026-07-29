# AI4SE Coding Agent Harness · WebUI Design System

This specification adapts the attached Linear system into a fixed light-theme, dense developer-tool interface for the frozen `/api/v1` contract. It is implementation-ready for React 19.2, TypeScript, and Vite 8.1 without changing the product’s safety model.

## 1. Color

- The theme is fixed light; there is no dark-mode switch and no automatic color-scheme negotiation.
- Use the attached Linear tokens without introducing a second palette. The light role mapping is: `--fg` for the page canvas, `--accent-on` for raised panels, `--bg` for primary text and dark primary actions, `--meta` for supporting text, and token-derived `color-mix()` values for separators and quiet surfaces.
- `--accent` is reserved for the active navigation item, keyboard focus, and the single highest-priority action in a local decision context. Avoid decorative accent use.
- Status meaning never depends on color. Every status contains a visible symbol and text, for example `✓ Succeeded`, `! Approval required`, `× Failed`, or `◇ SIMULATED`.
- Success, warning, and danger tokens may tint a status symbol or bounded state background only. Body text and interactive labels must retain WCAG 2.2 AA contrast.
- Decorative gradients are prohibited. Charts use solid fills, quiet rules, and direct labels.

## 2. Typography

- Use `Inter Variable` for UI copy with `"cv01"` and `"ss03"` enabled globally. Use Berkeley Mono or the defined monospace fallback for run IDs, event types, timestamps, hashes, budgets, code, and machine-readable reasons.
- Weight vocabulary is limited to 400 for reading, 510 for controls and emphasis, and 590 for headings and urgent labels. Do not use 700.
- Default UI text is 14px with a 1.5 line height. Metadata and dense table labels use 12px; form inputs and long explanations use 14–16px.
- Page view headings use 32px/1.13 with Linear’s negative tracking. Section and panel headings remain compact at 14–18px.
- All generated or user-provided strings must wrap safely. Run IDs, repository paths, hashes, and event payloads use `overflow-wrap: anywhere` or a bounded horizontal scroller where character alignment is essential.

## 3. Spacing

- Use the provided 4, 8, 12, 16, 20, 24, 32, and 48px spacing tokens. The dashboard’s default rhythm is intentionally dense: 8–12px within a group, 20–24px between functional groups.
- Card padding is 16px by default and 20px for forms. Table rows use 12px vertical padding. Panel headers have a minimum height of 52px.
- Controls have a minimum 44px target in both dimensions even when the visible glyph is smaller.
- Common-region borders are used only where they clarify an operational unit: KPI card, chart, form group, event stream, approval panel, or credential record.
- Mobile spacing collapses to a 12px page gutter and single-column flow without shrinking touch targets.

## 4. Layout

- Desktop uses a 240px fixed/sticky sidebar, a 64px sticky top bar, and an independently scrolling main region. The main content canvas may grow to 1500px to preserve dense tables and evidence panes.
- Five primary surfaces share one shell: Dashboard/recent runs, New Run/preflight, Run Detail, Credentials, and Demo Gallery.
- Dashboard begins with four KPI cards, then primary run-volume visualization plus live activity, then the recent-runs table.
- New Run uses a main form and a narrower preflight rail. Run creation is unavailable until native validation passes.
- Run Detail uses a wide evidence column and a narrow control/status rail. Timeline, bounded evidence, diff, budgets, approval, SSE coverage, and terminal reason remain distinct, named regions.
- At narrower widths, two-column areas become one column. At 760px, primary navigation becomes a horizontally scrollable tab row and all data regions remain operable without page-level horizontal overflow.
- The demo surface is visually and semantically isolated. A persistent banner and every demo card/action must say `SIMULATED`.

## 5. Components

- `AppShell`: owns sidebar, top bar, global search shortcut, current surface, and focus transfer after navigation.
- `StatusLabel`: always renders `{icon, text, tone}`. Icon-only statuses are invalid.
- `KpiCard`, `Panel`, `DataTable`, and `MetricChart`: compose the dashboard. Charts use inline SVG with a text title and description; no charting runtime is required.
- `PreflightForm`: uses native labels, inputs, selects, constraint validation, and a submitted-pending state. Errors preserve all input and focus the first invalid control.
- `EventTimeline`: renders typed `/api/v1` events in cursor order with timestamp, event type, and plain-language payload summary.
- `BudgetMeter`: provides visible numeric remaining/total values plus native `role="meter"` semantics.
- `EvidenceViewer`: caps its height and byte count, scrolls internally, labels truncation, and renders redactions as explicit `REDACTED` blocks. It never exposes credentials, authorization headers, or unbounded payloads.
- `DiffViewer`: is read-only, keyboard focusable, horizontally scrollable where needed, and never masquerades as an editable terminal. The product contains no embedded terminal.
- `ApprovalPanel`: offers exactly `Approve once` and `Reject`. A submitted decision disables both controls. Permanent allow, remember-this-choice, wildcard scope, and policy editing are prohibited.
- `CredentialCard`: accepts replacement material only through `type="password"` with appropriate autocomplete. Existing values are represented only by non-secret metadata such as a short ending; reveal and copy-secret controls do not exist.
- `ConnectionState`: covers disconnected, reconnecting, and reconnected states with icon, text, cursor behavior, and recovery detail. Live changes use a pre-existing polite live region.
- `TerminalReason`: records one server-provided terminal reason and explanatory copy. It is not inferred from client state.
- `DemoCard`: repeats `SIMULATED` in its label and action. Demo mode visibly omits arbitrary local run creation, credentials, config validation, and non-fixed artifacts.
- Every data-bearing component must support loading, empty, error, populated, and edge states. Loading receives a 15-second delayed notice; errors provide a cause and recovery; empty states provide an explanation and action.

## 6. Motion

- Motion is functional and brief: 150ms for hover/press feedback and up to 200ms for surface changes. Navigation itself does not animate spatially.
- SSE recovery may update status text in place; it must not pulse, flash, or use an indefinite spinner. Reconnection announces `Disconnected`, `Reconnecting`, then `Reconnected`.
- Submitted actions expose a pending label and disable duplicate submission. A request exceeding 15 seconds shows “Taking longer than expected”; work exceeding 60 seconds stops animation and offers recovery.
- Toasts are non-urgent, use a polite live region, stay in one consistent location, and can receive focus. Critical errors remain inline.
- `prefers-reduced-motion: reduce` removes non-essential transitions.

## 7. Voice

- Write in direct operational language: “Repository reachable,” “Approval required,” and “Stream closed; cursor retained.”
- Explain failures in the order: what happened, known cause, available recovery. Never use “Something went wrong.”
- Distinguish server fact from client inference. Use exact typed event names and terminal reason identifiers where they help debugging, paired with a human explanation.
- Safety language is explicit and bounded: “Approve this exact patch once,” “No permanent rule was created,” and “Secret values are never returned.”
- Avoid celebratory marketing language inside the harness. Success is calm, specific, and timestamped.
- All demo activity uses uppercase `SIMULATED` at the point of action and in the resulting detail context.

## 8. Brand

- The brand posture is precise, bounded, inspectable, and calm under failure. The visual identity comes from Linear’s geometric typography, fine separators, disciplined indigo, and engineered information density.
- The `A4` mark is compact and utilitarian. It does not compete with run state or approval decisions.
- Trust is expressed through visible constraints: frozen API version, policy profile, budgets, redaction, evidence hashes, read-only diffs, typed events, and explicit terminal reasons.
- Production and demo are separate modes, never stylistic variants of the same action. Demo must look useful while remaining visibly non-production.
- The implementation target is React 19.2 + TypeScript + Vite 8.1. Preserve semantic HTML beneath components, prefer native controls, and model API payloads as discriminated unions keyed by typed event names.
- Keep compressed production assets below 1.5 MiB: use no UI framework runtime, no chart library, inline only small SVGs, subset/self-host fonts where licensing permits, route-split secondary surfaces, and enforce the budget in CI.
- Accessibility targets zero serious or critical axe findings, full keyboard operation, logical focus order, visible 3px focus indicators, named landmarks, labeled inputs, accessible charts, and no color-only state.

## 9. Anti-patterns

- No dark theme, theme toggle, decorative gradient, warm cream canvas, glassmorphism, oversized marketing hero, or ornamental illustration.
- No embedded terminal, editable diff, fake command prompt, or “developer” styling that substitutes monospace for hierarchy.
- No permanent allow, “remember this decision,” global approval, wildcard approval scope, or approval detached from an evidence hash.
- No secret reveal, secret copy, plaintext credential value, populated password field, secret in logs, or secret echoed after save.
- No demo action that can create arbitrary local runs, accept credentials, validate arbitrary config, or produce non-fixed artifacts. No demo event may omit `SIMULATED`.
- No icon-only status, color-only error, vague failure copy, hidden focus ring, positive `tabindex`, click-only custom control, or unlabeled input.
- No infinite spinner, auto-cleared failed form, first-keystroke validation, destructive toast, or connection state without cursor/recovery context.
- No unbounded evidence, raw authorization material, unconstrained event payload, hot-linked asset, third-party charting dependency, or production bundle above 1.5 MiB compressed.
- No lorem ipsum, invented vanity metrics, ambiguous placeholders, or product claims unsupported by the frozen `/api/v1` contract.
