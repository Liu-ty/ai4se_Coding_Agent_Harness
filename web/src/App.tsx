import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiClient, RunEventStream } from "./api/client";
import type {
  ConnectionState,
  CreateRunRequest,
  Run,
  RunEvent,
  RuntimeConfig,
} from "./api/types";
import type { ApprovalRequest } from "./components/ApprovalPanel";
import { Credentials } from "./pages/Credentials";
import { Dashboard, type DashboardState } from "./pages/Dashboard";
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

function defaultRuntime(): RuntimeConfig {
  return window.__AI4SE_RUNTIME__ ?? {
    csrfToken: undefined,
    capabilities: location.search.includes("fixture=demo")
      ? { ...localCapabilities, createRuns: false, cancelRuns: false, approvals: false,
        artifacts: false, configValidation: false, credentials: false, demo: true,
        fixedRuns: ["feedback-loop"] }
      : localCapabilities,
  };
}

type Page = "dashboard" | "new-run" | "run-detail" | "credentials" | "demos";

function approvalFrom(events: RunEvent[]): ApprovalRequest | undefined {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const payload = events[index].payload;
    if (typeof payload.digest !== "string") continue;
    return {
      digest: payload.digest,
      action: typeof payload.action === "string" ? payload.action : "Unknown action",
      files: Array.isArray(payload.files) ? payload.files.filter((item): item is string =>
        typeof item === "string") : [],
      risk: typeof payload.risk === "string" ? payload.risk : "Server risk detail unavailable",
    };
  }
}

function artifactReference(event: RunEvent) {
  for (const key of ["diff_artifact_id", "artifact_id"]) {
    if (typeof event.payload[key] === "string") return event.payload[key] as string;
  }
}

export function App({ runtimeConfig, apiClient }: {
  runtimeConfig?: RuntimeConfig;
  apiClient?: ApiClient;
} = {}) {
  const runtime = useMemo(() => runtimeConfig ?? defaultRuntime(), [runtimeConfig]);
  const client = useMemo(() => apiClient ?? new ApiClient({ csrfToken: runtime.csrfToken }),
    [apiClient, runtime.csrfToken]);
  const [page, setPage] = useState<Page>(runtime.capabilities.demo ? "demos" : "dashboard");
  const [runs, setRuns] = useState<Run[]>([]);
  const [dashboardState, setDashboardState] = useState<DashboardState>("loading");
  const [dashboardError, setDashboardError] = useState("");
  const [selected, setSelected] = useState<Run>();
  const [detailLoading, setDetailLoading] = useState(false);
  const [events, setEvents] = useState<RunEvent[]>([]);
  const [connection, setConnection] = useState<ConnectionState>("disconnected");
  const [diff, setDiff] = useState<{ content: string; truncated: boolean }>();
  const [actionError, setActionError] = useState("");
  const [cancelPending, setCancelPending] = useState(false);
  const streamRef = useRef<RunEventStream | undefined>(undefined);

  const navigate = useCallback((next: Page) => {
    if (next !== "run-detail") {
      streamRef.current?.stop();
      streamRef.current = undefined;
    }
    setPage(next);
    requestAnimationFrame(() => document.querySelector<HTMLElement>("#main-content")?.focus());
  }, []);

  const loadRuns = useCallback(async () => {
    setDashboardState("loading");
    setDashboardError("");
    const delayed = window.setTimeout(() => setDashboardState("delayed"), 15_000);
    try {
      const result = await client.listRuns(0, 50);
      setRuns(result.runs);
      setDashboardState(result.runs.length ? "populated" : "empty");
    } catch (error) {
      setDashboardState("error");
      setDashboardError(error instanceof Error ? error.message : "Run page request failed.");
    } finally {
      window.clearTimeout(delayed);
    }
  }, [client]);

  useEffect(() => { void loadRuns(); }, [loadRuns]);
  useEffect(() => () => streamRef.current?.stop(), []);

  const loadArtifact = useCallback(async (runId: string, artifactId: string) => {
    if (!runtime.capabilities.artifacts) return;
    try {
      const artifact = await client.artifact(runId, artifactId);
      if (artifact.kind.toLowerCase().includes("diff")) {
        setDiff({ content: artifact.content, truncated: artifact.truncated });
      }
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Artifact could not be loaded.");
    }
  }, [client, runtime.capabilities.artifacts]);

  const openRun = useCallback(async (runId: string) => {
    streamRef.current?.stop();
    setDetailLoading(true);
    setSelected(undefined);
    setEvents([]);
    setDiff(undefined);
    setActionError("");
    navigate("run-detail");
    try {
      const run = await client.getRun(runId);
      setSelected(run);
      const seenArtifacts = new Set<string>();
      const stream = new RunEventStream({
        onEvent: (event) => {
          setEvents((current) => current.some((item) => item.sequence === event.sequence)
            ? current : [...current, event].sort((left, right) => left.sequence - right.sequence));
          if (runtime.capabilities.demo && typeof event.payload.diff === "string") {
            setDiff({
              content: event.payload.diff,
              truncated: event.payload.truncated === true,
            });
          }
          const artifactId = artifactReference(event);
          if (artifactId && !seenArtifacts.has(artifactId)) {
            seenArtifacts.add(artifactId);
            void loadArtifact(runId, artifactId);
          }
        },
        onState: setConnection,
      });
      streamRef.current = stream;
      void stream.run(`/api/v1/runs/${encodeURIComponent(runId)}/events`);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Run could not be loaded.");
    } finally {
      setDetailLoading(false);
    }
  }, [client, loadArtifact, navigate]);

  const createRun = async (request: CreateRunRequest) => {
    const run = await client.createRun(request);
    await loadRuns();
    await openRun(run.id);
  };
  const decide = async (decision: "approve" | "reject", digest: string) => {
    if (!selected) return;
    setActionError("");
    try {
      if (decision === "approve") await client.approve(selected.id, digest);
      else await client.reject(selected.id, digest, false);
      setEvents((current) => current.filter((event) => event.payload.digest !== digest));
      setSelected(await client.getRun(selected.id));
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Approval decision failed.");
    }
  };
  const cancel = async () => {
    if (!selected) return;
    setCancelPending(true);
    setActionError("");
    try {
      await client.cancel(selected.id);
      setSelected(await client.getRun(selected.id));
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Cancellation failed.");
    } finally {
      setCancelPending(false);
    }
  };

  const approval = runtime.capabilities.approvals ? approvalFrom(events) : undefined;
  const demoRuns = runs.map((run) => run.id);

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
      <span>{runtime.capabilities.demo ? "SIMULATED" : "Authenticated local session"}</span>
      <span>Fixed light theme</span>
    </header>
    <main id="main-content" tabIndex={-1}>
      {page === "dashboard" && <Dashboard runs={runs} state={dashboardState}
        error={dashboardError} onRetry={() => void loadRuns()} onOpen={(id) => void openRun(id)} />}
      {page === "new-run" && <NewRun onPreflight={(request) => client.preflight(request)}
        onCreate={createRun} />}
      {page === "credentials" && <Credentials
        loadStatus={(provider, host) => client.credentialStatus(provider, host)}
        onSave={(provider, host, secret) => client.saveCredential(provider, host, secret)}
        onClear={(provider, host) => client.clearCredential(provider, host)} />}
      {page === "demos" && (dashboardState === "loading" || dashboardState === "delayed") &&
        <section className="panel"><h1>Demo Gallery</h1><p>Loading fixed SIMULATED scenarios…</p></section>}
      {page === "demos" && dashboardState === "error" && <section className="panel" role="alert">
        <h1>Demo Gallery</h1><p>{dashboardError}</p><button onClick={() => void loadRuns()}>Retry demos</button></section>}
      {page === "demos" && (dashboardState === "empty" || dashboardState === "populated") &&
        <DemoGallery fixedRuns={demoRuns} onOpen={(id) => void openRun(id)} />}
      {page === "run-detail" && detailLoading && <section className="panel"><h1>Run Detail</h1>
        <p>Loading the server snapshot and event cursor…</p></section>}
      {page === "run-detail" && !detailLoading && !selected && <section className="panel" role="alert">
        <h1>Run Detail unavailable</h1><p>{actionError || "No run was selected."}</p></section>}
      {page === "run-detail" && selected && <RunDetail run={selected} events={events}
        simulated={runtime.capabilities.demo} connection={connection} diff={diff}
        approval={approval} onDecision={(decision, digest) => void decide(decision, digest)}
        onCancel={runtime.capabilities.cancelRuns ? () => void cancel() : undefined}
        cancelPending={cancelPending} error={actionError} />}
    </main></div>
  </div>;
}
