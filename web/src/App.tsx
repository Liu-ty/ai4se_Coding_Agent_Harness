import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiClient, RunEventStream } from "./api/client";
import type {
  ConnectionState,
  CreateRunRequest,
  Run,
  RunEvent,
  RuntimeConfig,
  StreamFailure,
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
  const fixture = new URLSearchParams(location.search).get("fixture");
  return window.__AI4SE_RUNTIME__ ?? {
    csrfToken: undefined,
    capabilities: fixture === "demo"
      ? { ...localCapabilities, createRuns: false, cancelRuns: false, approvals: false,
        artifacts: false, configValidation: false, credentials: false, demo: true,
        fixedRuns: ["feedback-loop"] }
      : localCapabilities,
  };
}

type Page = "dashboard" | "new-run" | "run-detail" | "credentials" | "demos";

function approvalFrom(events: RunEvent[], decidedDigests: ReadonlySet<string>): ApprovalRequest | undefined {
  const closingEvents = new Set([
    "ApprovalGranted", "ApprovalRejected", "ReviewComplete", "RunRecovered", "RunStopped",
    "RunSucceeded",
  ]);
  let approval: ApprovalRequest | undefined;
  for (const event of [...events].sort((left, right) => left.sequence - right.sequence)) {
    if (closingEvents.has(event.type)) {
      approval = undefined;
      continue;
    }
    if (event.type !== "ApprovalRequired") continue;
    const payload = event.payload;
    const action = payload.action;
    if (typeof payload.digest !== "string" || !action || typeof action !== "object" ||
      Array.isArray(action) || typeof (action as Record<string, unknown>).kind !== "string") continue;
    if (decidedDigests.has(payload.digest)) continue;
    const actionRecord = action as Record<string, unknown>;
    approval = {
      digest: payload.digest,
      action: {
        kind: actionRecord.kind as string,
        args: actionRecord.args && typeof actionRecord.args === "object" &&
          !Array.isArray(actionRecord.args)
          ? actionRecord.args as Record<string, unknown> : {},
      },
      affectedFiles: Array.isArray(payload.affected_files)
        ? payload.affected_files.filter((item): item is string =>
        typeof item === "string") : [],
      risk: typeof payload.risk === "string" ? payload.risk : "unknown",
      riskReason: typeof payload.risk_reason === "string"
        ? payload.risk_reason : "Server risk detail unavailable",
    };
  }
  return approval;
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
  const [decidedDigests, setDecidedDigests] = useState<Set<string>>(() => new Set());
  const [connection, setConnection] = useState<ConnectionState>("disconnected");
  const [streamError, setStreamError] = useState<StreamFailure>();
  const [diff, setDiff] = useState<{ content: string; truncated: boolean }>();
  const [actionError, setActionError] = useState("");
  const [cancelPending, setCancelPending] = useState<{
    runId: string;
    selectionGeneration: number;
  }>();
  const streamRef = useRef<RunEventStream | undefined>(undefined);
  const streamURLRef = useRef("");
  const selectionGenerationRef = useRef(0);
  const snapshotGenerationRef = useRef(0);
  const reconnectPendingRef = useRef(false);

  const navigate = useCallback((next: Page) => {
    if (next !== "run-detail") {
      selectionGenerationRef.current += 1;
      setCancelPending(undefined);
      streamRef.current?.stop();
      streamRef.current = undefined;
      streamURLRef.current = "";
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

  const refreshSnapshot = useCallback(async (runId: string, selectionGeneration: number) => {
    const snapshotGeneration = ++snapshotGenerationRef.current;
    const run = await client.getRun(runId);
    if (selectionGeneration === selectionGenerationRef.current &&
      snapshotGeneration === snapshotGenerationRef.current) {
      setSelected(run);
    }
    return run;
  }, [client]);

  const loadArtifact = useCallback(async (
    runId: string,
    artifactId: string,
    selectionGeneration: number,
  ) => {
    if (!runtime.capabilities.artifacts) return;
    try {
      const artifact = await client.artifact(runId, artifactId);
      if (selectionGeneration === selectionGenerationRef.current &&
        artifact.kind.toLowerCase().includes("diff")) {
        setDiff({ content: artifact.content, truncated: artifact.truncated });
      }
    } catch (error) {
      if (selectionGeneration === selectionGenerationRef.current) {
        setActionError(error instanceof Error ? error.message : "Artifact could not be loaded.");
      }
    }
  }, [client, runtime.capabilities.artifacts]);

  const openRun = useCallback(async (runId: string) => {
    const selectionGeneration = ++selectionGenerationRef.current;
    setCancelPending(undefined);
    streamRef.current?.stop();
    setDetailLoading(true);
    setSelected(undefined);
    setEvents([]);
    setDecidedDigests(new Set());
    setDiff(undefined);
    setActionError("");
    setStreamError(undefined);
    setConnection("disconnected");
    navigate("run-detail");
    try {
      await refreshSnapshot(runId, selectionGeneration);
      if (selectionGeneration !== selectionGenerationRef.current) return;
      const seenArtifacts = new Set<string>();
      const stream = new RunEventStream({
        onEvent: (event) => {
          if (selectionGeneration !== selectionGenerationRef.current) return;
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
            void loadArtifact(runId, artifactId, selectionGeneration);
          }
          void refreshSnapshot(runId, selectionGeneration).catch((error) => {
            if (selectionGeneration === selectionGenerationRef.current) {
              setActionError(error instanceof Error ? error.message : "Run snapshot could not be refreshed.");
            }
          });
        },
        onState: (state) => {
          if (selectionGeneration === selectionGenerationRef.current) setConnection(state);
        },
        onError: (failure) => {
          if (selectionGeneration === selectionGenerationRef.current) setStreamError(failure);
        },
      });
      streamRef.current = stream;
      streamURLRef.current = `/api/v1/runs/${encodeURIComponent(runId)}/events`;
      void stream.run(streamURLRef.current);
    } catch (error) {
      if (selectionGeneration === selectionGenerationRef.current) {
        setActionError(error instanceof Error ? error.message : "Run could not be loaded.");
      }
    } finally {
      if (selectionGeneration === selectionGenerationRef.current) setDetailLoading(false);
    }
  }, [loadArtifact, navigate, refreshSnapshot, runtime.capabilities.demo]);

  const createRun = async (request: CreateRunRequest) => {
    const run = await client.createRun(request);
    await loadRuns();
    await openRun(run.id);
  };
  const decide = async (decision: "approve" | "reject", digest: string) => {
    if (!selected) return;
    const runId = selected.id;
    const selectionGeneration = selectionGenerationRef.current;
    setActionError("");
    try {
      if (decision === "approve") await client.approve(runId, digest);
      else await client.reject(runId, digest, false);
      setDecidedDigests((current) => new Set(current).add(digest));
      await refreshSnapshot(runId, selectionGeneration);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Approval decision failed.");
      throw error;
    }
  };
  const cancel = async () => {
    if (!selected) return;
    const runId = selected.id;
    const selectionGeneration = selectionGenerationRef.current;
    setCancelPending({ runId, selectionGeneration });
    setActionError("");
    try {
      await client.cancel(runId);
      if (selectionGeneration !== selectionGenerationRef.current) return;
      await refreshSnapshot(runId, selectionGeneration);
    } catch (error) {
      if (selectionGeneration === selectionGenerationRef.current) {
        setActionError(error instanceof Error ? error.message : "Cancellation failed.");
      }
    } finally {
      if (selectionGeneration === selectionGenerationRef.current) setCancelPending(undefined);
    }
  };

  const approval = runtime.capabilities.approvals && selected?.state === "AWAITING_APPROVAL"
    ? approvalFrom(events, decidedDigests) : undefined;
  const declaredDemoRuns = new Set(runtime.capabilities.fixedRuns);
  const demoRuns = runs.filter((run) => declaredDemoRuns.has(run.id)).map((run) => run.id);
  const reconnect = () => {
    if (!streamRef.current || !streamURLRef.current || reconnectPendingRef.current) return;
    reconnectPendingRef.current = true;
    setStreamError(undefined);
    setConnection("reconnecting");
    void streamRef.current.run(streamURLRef.current).finally(() => {
      reconnectPendingRef.current = false;
    });
  };

  return <div className="shell">
    <aside className="sidebar"><a className="brand" href="#" onClick={(event) => {
      event.preventDefault(); navigate(runtime.capabilities.demo ? "demos" : "dashboard");
    }}><span aria-hidden="true">A4</span> AI4SE Harness</a>
      <nav aria-label="Primary">
        {!runtime.capabilities.demo && <button aria-current={page === "dashboard" ? "page" : undefined}
          onClick={() => navigate("dashboard")}>▦ Dashboard</button>}
        {runtime.capabilities.createRuns && <button aria-current={page === "new-run" ? "page" : undefined}
          onClick={() => navigate("new-run")}>＋ New Run</button>}
        {runtime.capabilities.credentials && <button aria-current={page === "credentials" ? "page" : undefined}
          onClick={() => navigate("credentials")}>⌁ Credentials</button>}
        <button aria-current={page === "demos" ? "page" : undefined}
          onClick={() => navigate("demos")}>◇ Demo Gallery</button>
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
      {page === "new-run" && <NewRun
        onPreflight={(request, signal) => client.preflight(request, signal)}
        onCreate={createRun} />}
      {page === "credentials" && <Credentials
        loadStatus={(provider, host, signal) => client.credentialStatus(provider, host, signal)}
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
        approval={approval} onDecision={decide}
        onCancel={runtime.capabilities.cancelRuns ? () => void cancel() : undefined}
        cancelPending={cancelPending?.runId === selected.id &&
          cancelPending.selectionGeneration === selectionGenerationRef.current} error={actionError}
        streamError={streamError} onReconnect={reconnect} />}
    </main></div>
  </div>;
}
