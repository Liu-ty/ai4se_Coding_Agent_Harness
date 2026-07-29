import type { Run } from "../api/types";
import { StatusLabel } from "../components/StatusLabel";

export function Dashboard({ runs, onOpen }: { runs: Run[]; onOpen: (id: string) => void }) {
  return <section><header className="page-head"><div><h1>Dashboard</h1>
    <p>Recent bounded runs and their objective terminal state.</p></div></header>
    <div className="cards">
      <article className="panel metric"><span>Visible runs</span><strong>{runs.length}</strong></article>
      <article className="panel metric"><span>Active</span>
        <strong>{runs.filter((run) => !["SUCCEEDED", "STOPPED", "REVIEW_COMPLETE"].includes(run.state)).length}</strong>
      </article>
    </div>
    <section className="panel"><h2>Recent runs</h2>
      {runs.length === 0 ? <p>No run has been opened in this session.</p> :
        <table><thead><tr><th>Run</th><th>Repository</th><th>Status</th></tr></thead>
          <tbody>{runs.map((run) => <tr key={run.id}><td><button className="link"
            onClick={() => onOpen(run.id)}>{run.id}</button></td><td>{run.repo_root}</td>
            <td><StatusLabel text={run.state} tone={run.state === "SUCCEEDED" ? "success" : "neutral"} /></td></tr>)}</tbody>
        </table>}
    </section>
  </section>;
}
