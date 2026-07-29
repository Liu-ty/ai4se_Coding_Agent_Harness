import { useMemo, useState } from "react";
import { ApiClient } from "./api/client";
import type { CreateRunRequest, Run, RunEvent, RuntimeConfig } from "./api/types";
import { Credentials } from "./pages/Credentials";
import { Dashboard } from "./pages/Dashboard";
import { DemoGallery } from "./pages/DemoGallery";
import { NewRun } from "./pages/NewRun";
import { RunDetail } from "./pages/RunDetail";

declare global {
  interface Window { __AI4SE_RUNTIME__?: RuntimeConfig }
}

const localCapabilities = {
  createRuns: true, cancelRuns: true, approvals: true, artifacts: true,
  configValidation: true, credentials: true, demo: false, fixedRuns: [],
};
const runtime = window.__AI4SE_RUNTIME__ ?? {
  csrfToken: undefined,
  capabilities: location.search.includes("demo=1")
    ? { ...localCapabilities, createRuns: false, cancelRuns: false, approvals: false,
      artifacts: false, configValidation: false, credentials: false, demo: true,
      fixedRuns: ["feedback-loop"] }
    : localCapabilities,
};

type Page = "dashboard" | "new-run" | "run-detail" | "credentials" | "demos";

const now = "2026-07-29T08:00:00Z";
const demoRun: Run = {
  id: "feedback-loop", state: "SUCCEEDED", profile: "workspace-auto",
  task: "Repair the deterministic addition defect", repo_root: "SIMULATED/workspace",
  current_stage: "final", created_at: now, updated_at: now,
};
const demoEvents: RunEvent[] = [
  { sequence: 1, type: "PolicyEvaluated", payload: {
    category: "POLICY_DENIED", summary: "Guardrail intercepted a protected-path patch.",
  }},
  { sequence: 2, type: "ValidationFailed", payload: {
    category: "TEST_FAILURE", summary: "Injected failure: expected 2, received 1.",
    evidence: [{ source: "stderr", message: "REDACTED", path: "sum.test.ts", line: 12 }],
    budgets: { mutations: { used: 1, limit: 5 } },
  }},
  { sequence: 3, type: "DecisionChanged", payload: {
    category: "ACTION_CHANGED", summary: "Feedback changed the second patch to return a + b.",
  }},
  { sequence: 4, type: "RunSucceeded", payload: {
    category: "SUCCEEDED", summary: "Every required final validation stage passed.",
  }},
];

export function App() {
  const client = useMemo(() => new ApiClient({ csrfToken: runtime.csrfToken }), []);
  const [page, setPage] = useState<Page>(runtime.capabilities.demo ? "demos" : "dashboard");
  const [runs, setRuns] = useState<Run[]>([]);
  const [selected, setSelected] = useState<Run | undefined>();
  const [approval, setApproval] = useState(false);

  const navigate = (next: Page) => { setPage(next); requestAnimationFrame(() =>
    document.querySelector<HTMLElement>("#main-content")?.focus()); };
  const createDraft = async (request: CreateRunRequest) => {
    const draft: Run = {
      id: "draft-supervised", state: "AWAITING_APPROVAL", profile: request.profile,
      task: request.task, repo_root: request.repo_root, current_stage: "preflight",
      created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
    };
    setRuns((value) => [draft, ...value]);
    setSelected(draft);
    setApproval(request.profile === "supervised");
    navigate("run-detail");
  };
  const openDemo = () => { setSelected(demoRun); setApproval(false); navigate("run-detail"); };

  return <div className="shell">
    <aside className="sidebar"><a className="brand" href="#" onClick={(event) => {
      event.preventDefault(); navigate(runtime.capabilities.demo ? "demos" : "dashboard");
    }}><span aria-hidden="true">A4</span> AI4SE Harness</a>
      <nav aria-label="Primary">
        {!runtime.capabilities.demo && <button onClick={() => navigate("dashboard")}>▦ Dashboard</button>}
        {runtime.capabilities.createRuns && <button onClick={() => navigate("new-run")}>＋ New Run</button>}
        {runtime.capabilities.credentials && <button onClick={() => navigate("credentials")}>⌁ Credentials</button>}
        <button onClick={() => navigate("demos")}>◇ Demo Gallery</button>
      </nav>
      <small>API v1 · bounded local interface</small>
    </aside>
    <div className="workspace"><header className="topbar">
      <span>{runtime.capabilities.demo ? "SIMULATED" : "Local session"}</span>
      <span>Fixed light theme</span>
    </header>
    <main id="main-content" tabIndex={-1}>
      {page === "dashboard" && <Dashboard runs={runs} onOpen={(id) => {
        setSelected(runs.find((run) => run.id === id)); navigate("run-detail");
      }} />}
      {page === "new-run" && <NewRun onCreate={createDraft} />}
      {page === "credentials" && <Credentials onSave={(provider, host, secret) =>
        client.saveCredential(provider, host, secret)} />}
      {page === "demos" && <DemoGallery fixedRuns={runtime.capabilities.fixedRuns.length
        ? runtime.capabilities.fixedRuns : ["feedback-loop"]} onOpen={openDemo} />}
      {page === "run-detail" && selected && <RunDetail run={selected}
        simulated={selected.id === demoRun.id}
        events={selected.id === demoRun.id ? demoEvents : []}
        diff={selected.id === demoRun.id ? {
          content: "--- a/sum.ts\n+++ b/sum.ts\n- return 1\n+ return a + b",
          truncated: false,
        } : undefined}
        approval={approval ? {
          digest: "sha256:draft-supervised", action: "apply_patch",
          files: ["src/sum.ts"], risk: "Modifies one tracked source file",
        } : undefined}
        onDecision={() => setApproval(false)} />}
    </main></div>
  </div>;
}
