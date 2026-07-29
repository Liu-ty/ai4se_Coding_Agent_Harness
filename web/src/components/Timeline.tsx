import { useState } from "react";
import type { RunEvent } from "../api/types";
import { StatusLabel } from "./StatusLabel";

type Budget = { used: number; limit: number };
type Evidence = { source?: string; message?: string; path?: string; line?: number };

export function Timeline({ events }: { events: RunEvent[] }) {
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});
  return <section className="panel" aria-labelledby="timeline-heading">
    <h2 id="timeline-heading">Event timeline</h2>
    <ol className="timeline">
      {[...events].sort((a, b) => a.sequence - b.sequence).map((event) => {
        const category = typeof event.payload.category === "string" ? event.payload.category : undefined;
        const summary = typeof event.payload.summary === "string" ? event.payload.summary : event.type;
        const evidence = Array.isArray(event.payload.evidence) ? event.payload.evidence as Evidence[] : [];
        const budgets = event.payload.budgets as { mutations?: Budget } | undefined;
        return <li key={event.sequence}>
          <div className="event-meta"><code>#{event.sequence} {event.type}</code></div>
          {category && <StatusLabel text={category} tone={category.includes("FAIL") ? "danger" : "neutral"} />}
          <p>{summary}</p>
          {budgets?.mutations &&
            <div className="budget" role="meter" aria-label="Mutation budget"
              aria-valuemin={0} aria-valuemax={budgets.mutations.limit}
              aria-valuenow={budgets.mutations.used}>
              Mutations {budgets.mutations.used} / {budgets.mutations.limit}
            </div>}
          {evidence.length > 0 && <>
            <button className="quiet" onClick={() => setExpanded((value) => ({
              ...value, [event.sequence]: !value[event.sequence],
            }))}>{expanded[event.sequence] ? "Hide redacted evidence" : "Show redacted evidence"}</button>
            {expanded[event.sequence] &&
              <pre className="evidence">{evidence.slice(0, 8).map((item, index) =>
                <span key={index}>{item.path ?? item.source ?? "evidence"}{item.line ? `:${item.line}` : ""}{" "}
                  <span>{item.message ?? ""}</span>{"\n"}</span>)}</pre>}
          </>}
        </li>;
      })}
    </ol>
  </section>;
}
