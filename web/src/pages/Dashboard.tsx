import type { Run } from "../api/types";
import { StatusLabel } from "../components/StatusLabel";

export type DashboardState = "loading" | "delayed" | "error" | "empty" | "populated";

const terminalStates = new Set(["SUCCEEDED", "STOPPED", "REVIEW_COMPLETE"]);

export function Dashboard({ runs, state, error, onOpen, onRetry }: {
  runs: Run[];
  state: DashboardState;
  error?: string;
  onOpen: (id: string) => void;
  onRetry: () => void;
}) {
  const active = runs.filter((run) => !terminalStates.has(run.state) &&
    run.state !== "AWAITING_APPROVAL").length;
  const approvals = runs.filter((run) => run.state === "AWAITING_APPROVAL").length;
  const terminal = runs.filter((run) => terminalStates.has(run.state)).length;
  const chartMaximum = Math.max(active, approvals, terminal, 1);
  const bar = (count: number) => {
    const height = (count / chartMaximum) * 80;
    return { y: 100 - height, height };
  };
  const metrics = [
    ["Visible runs", runs.length], ["Active", active],
    ["Awaiting approval", approvals], ["Terminal", terminal],
  ] as const;
  return <section><header className="page-head"><div><h1>Dashboard</h1>
    <p>Bounded server run snapshots and recent objective activity.</p></div>
    <button onClick={onRetry}>Refresh runs</button></header>
    {(state === "loading" || state === "delayed") && <section className="panel" aria-live="polite">
      <h2>Loading recent runs</h2><p>{state === "delayed" ?
        "Taking longer than expected. The server may still be recovering." :
        "Requesting the first bounded page from the local server…"}</p></section>}
    {state === "error" && <section className="panel error-panel" role="alert">
      <h2>Recent runs unavailable</h2><p>{error || "The server did not return the run page."}</p>
      <button onClick={onRetry}>Retry loading runs</button></section>}
    {(state === "empty" || state === "populated") && <>
      <div className="kpi-grid">{metrics.map(([label, value]) =>
        <article data-testid="kpi-card" className="panel metric" key={label}>
          <span>{label}</span><strong>{value}</strong></article>)}</div>
      <div className="dashboard-grid">
        <section className="panel"><h2>Run state distribution</h2>
          <svg role="img" aria-label="Run state distribution" viewBox="0 0 400 120">
            <title>Run state distribution</title>
            <desc>{active} active, {approvals} awaiting approval, {terminal} terminal.</desc>
            <rect x="20" {...bar(active)} width="80" />
            <rect x="150" {...bar(approvals)} width="80" />
            <rect x="280" {...bar(terminal)} width="80" />
            <text x="20" y="116">Active</text><text x="150" y="116">Approval</text>
            <text x="280" y="116">Terminal</text>
          </svg>
        </section>
        <section className="panel" role="region" aria-label="Live activity"><h2>Live activity</h2>
          {runs.length === 0 ? <p>No activity is available.</p> :
            <ul className="activity-list">{runs.slice(0, 5).map((run) => <li key={run.id}>
              <StatusLabel text={run.state} tone={run.state === "SUCCEEDED" ? "success" : "neutral"} />
              <span>{run.id} · {run.updated_at || "Timestamp unavailable"}</span></li>)}</ul>}
        </section>
      </div>
      <section className="panel"><h2>Recent runs</h2>
        {runs.length === 0 ? <p>No runs were returned. Create a bounded run to begin.</p> :
          <div className="table-scroll"><table><thead><tr><th>Run</th><th>Repository</th>
            <th>Stage</th><th>Status</th></tr></thead>
            <tbody>{runs.map((run) => <tr key={run.id}><td><button className="link"
              onClick={() => onOpen(run.id)}>{run.id}</button></td><td>{run.repo_root}</td>
              <td>{run.current_stage || "Not started"}</td><td><StatusLabel text={run.state}
                tone={run.state === "SUCCEEDED" ? "success" : "neutral"} /></td></tr>)}</tbody>
          </table></div>}
      </section>
    </>}
  </section>;
}
