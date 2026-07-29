import type { ConnectionState, Run, RunEvent } from "../api/types";
import { ApprovalPanel, type ApprovalRequest } from "../components/ApprovalPanel";
import { DiffViewer } from "../components/DiffViewer";
import { StatusLabel } from "../components/StatusLabel";
import { Timeline } from "../components/Timeline";

type BudgetValue = { used: number; limit: number };
type Budgets = { decisions?: BudgetValue; mutations?: BudgetValue };

function latestBudgets(events: RunEvent[]): Budgets | undefined {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const value = events[index].payload.budgets;
    if (value && typeof value === "object") return value as Budgets;
  }
}

function latestReason(events: RunEvent[]) {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const reason = events[index].payload.reason;
    if (typeof reason === "string" && reason) return reason;
  }
}

export function RunDetail({ run, events, diff, approval, simulated, connection = "connected",
  onDecision, onCancel, cancelPending = false, error }: {
  run: Run;
  events: RunEvent[];
  diff?: { content: string; truncated: boolean };
  approval?: ApprovalRequest;
  simulated?: boolean;
  connection?: ConnectionState;
  onDecision?: (decision: "approve" | "reject", digest: string) => void;
  onCancel?: () => void;
  cancelPending?: boolean;
  error?: string;
}) {
  const budgets = latestBudgets(events);
  const reason = latestReason(events);
  return <section>
    {simulated && <div className="sim-banner">◇ SIMULATED — every row and artifact below is a fixed server fixture.</div>}
    <header className="page-head"><div><h1>{simulated ? "SIMULATED · " : ""}{run.id}</h1>
      <p>{run.task}</p></div><div className="actions">
        <StatusLabel text={run.state} tone={run.state === "SUCCEEDED" ? "success" : "neutral"} />
        {onCancel && <button className="danger" disabled={cancelPending} onClick={onCancel}>
          {cancelPending ? "Cancelling…" : "Cancel run"}</button>}
      </div></header>
    {error && <div className="panel error-panel" role="alert"><h2>Run action failed</h2><p>{error}</p></div>}
    <div className="detail-grid"><div className="stack">
      {events.length === 0 ? <section className="panel"><h2>Event timeline</h2>
        <p>No server event has been received yet.</p></section> :
        <Timeline events={events} simulated={simulated} />}
      {diff && <DiffViewer {...diff} />}</div>
      <aside className="stack"><section className="panel"><h2>Budgets</h2>
        {!budgets && <p>Budget data unavailable</p>}
        {budgets?.decisions && <div role="meter" aria-label="Decision budget" aria-valuemin={0}
          aria-valuemax={budgets.decisions.limit} aria-valuenow={budgets.decisions.used}>
          Decisions {budgets.decisions.used} / {budgets.decisions.limit}</div>}
        {budgets?.mutations && <div role="meter" aria-label="Mutation budget" aria-valuemin={0}
          aria-valuemax={budgets.mutations.limit} aria-valuenow={budgets.mutations.used}>
          Mutations {budgets.mutations.used} / {budgets.mutations.limit}</div>}
      </section>
      <section className="panel" aria-live="polite"><h2>Event stream</h2>
        <StatusLabel text={connection[0].toUpperCase() + connection.slice(1)}
          tone={connection === "disconnected" ? "danger" :
            connection === "reconnecting" ? "warning" : "success"} />
        <p>Latest sequence: {events.at(-1)?.sequence ?? 0}. Reconnect resumes from this cursor.</p>
      </section>
      {approval && onDecision && <ApprovalPanel request={approval} onDecision={onDecision} />}
      <section className="panel"><h2>Terminal reason</h2>
        {reason ? <code>{reason}</code> : <p>Terminal reason unavailable</p>}</section>
      </aside></div>
  </section>;
}
