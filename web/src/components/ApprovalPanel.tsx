import { useState } from "react";

export interface ApprovalRequest {
  digest: string;
  action: string;
  files: string[];
  risk: string;
}

export function ApprovalPanel({ request, onDecision }: {
  request: ApprovalRequest;
  onDecision: (decision: "approve" | "reject", digest: string) => void | Promise<void>;
}) {
  const [decided, setDecided] = useState(false);
  const decide = (decision: "approve" | "reject") => {
    if (decided) return;
    setDecided(true);
    void onDecision(decision, request.digest);
  };
  return <section className="panel approval-panel" aria-labelledby="approval-heading">
    <h2 id="approval-heading">Approval required</h2>
    <p><strong>Exact action:</strong> <code>{request.action}</code></p>
    <p><strong>Affected files:</strong> {request.files.join(", ")}</p>
    <p><strong>Risk:</strong> {request.risk}</p>
    <p><strong>Digest:</strong> <code>{request.digest}</code></p>
    <div className="actions">
      <button disabled={decided} onClick={() => decide("approve")}>Approve once</button>
      <button disabled={decided} className="danger" onClick={() => decide("reject")}>Reject</button>
    </div>
    <small>No permanent rule is created. This decision applies only to this exact digest.</small>
  </section>;
}
