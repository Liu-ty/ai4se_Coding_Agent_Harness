import type { ConnectionState, Run, RunEvent } from "../api/types";
import { ApprovalPanel, type ApprovalRequest } from "../components/ApprovalPanel";
import { DiffViewer } from "../components/DiffViewer";
import { StatusLabel } from "../components/StatusLabel";
import { Timeline } from "../components/Timeline";

export function RunDetail({ run, events, diff, approval, simulated, connection = "connected", onDecision }: {
  run: Run;
  events: RunEvent[];
  diff?: { content: string; truncated: boolean };
  approval?: ApprovalRequest;
  simulated?: boolean;
  connection?: ConnectionState;
  onDecision?: (decision: "approve" | "reject", digest: string) => void;
}) {
  return <section>
    {simulated && <div className="sim-banner">◇ SIMULATED — all events and artifacts on this run are fixed fixtures.</div>}
    <header className="page-head"><div><h1>{simulated ? "SIMULATED · " : ""}{run.id}</h1>
      <p>{run.task}</p></div><StatusLabel text={run.state} tone={run.state === "SUCCEEDED" ? "success" : "neutral"} /></header>
    <div className="detail-grid"><div className="stack"><Timeline events={events} />
      {diff && <DiffViewer {...diff} />}</div>
      <aside className="stack"><section className="panel"><h2>Budgets</h2>
        <div role="meter" aria-label="Decision budget" aria-valuemin={0} aria-valuemax={30} aria-valuenow={4}>Decisions 4 / 30</div>
        <div role="meter" aria-label="Mutation budget" aria-valuemin={0} aria-valuemax={5} aria-valuenow={2}>Mutations 2 / 5</div>
      </section>
      <section className="panel" aria-live="polite"><h2>Event stream</h2>
        <StatusLabel text={connection[0].toUpperCase() + connection.slice(1)}
          tone={connection === "disconnected" ? "danger" : connection === "reconnecting" ? "warning" : "success"} />
        <p>Latest sequence: {events.at(-1)?.sequence ?? 0}. Reconnect resumes from this cursor.</p>
      </section>
      {approval && onDecision && <ApprovalPanel request={approval} onDecision={onDecision} />}
      {["SUCCEEDED", "STOPPED", "REVIEW_COMPLETE"].includes(run.state) &&
        <section className="panel"><h2>Terminal reason</h2><code>{run.state}</code></section>}
      </aside></div>
  </section>;
}
