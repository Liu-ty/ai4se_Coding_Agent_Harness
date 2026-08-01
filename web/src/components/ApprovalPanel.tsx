import { useState } from "react";

export interface ApprovalRequest {
  digest: string;
  action: { kind: string; args: Record<string, unknown> };
  affectedFiles: string[];
  risk: string;
  riskReason: string;
}

export function ApprovalPanel({ request, onDecision }: {
  request: ApprovalRequest;
  onDecision: (decision: "approve" | "reject", digest: string) => Promise<void>;
}) {
  const [pending, setPending] = useState<"approve" | "reject">();
  const [decided, setDecided] = useState(false);
  const [error, setError] = useState("");
  const decide = async (decision: "approve" | "reject") => {
    if (decided || pending) return;
    setPending(decision);
    setError("");
    try {
      await onDecision(decision, request.digest);
      setDecided(true);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Approval decision failed. Retry.");
    } finally {
      setPending(undefined);
    }
  };
  return <section className="panel approval-panel" aria-labelledby="approval-heading">
    <h2 id="approval-heading">Approval required</h2>
    <p><strong>Exact action:</strong> <code>{request.action.kind}</code></p>
    <pre className="evidence">{JSON.stringify(request.action.args, null, 2)}</pre>
    <p><strong>Affected files:</strong> {request.affectedFiles.join(", ") || "None reported"}</p>
    <p><strong>Risk:</strong> {request.risk} — {request.riskReason}</p>
    <p><strong>Digest:</strong> <code>{request.digest}</code></p>
    {error && <p role="alert" className="notice">{error}</p>}
    <div className="actions">
      <button disabled={decided || Boolean(pending)} onClick={() => void decide("approve")}>
        {pending === "approve" ? "Approving…" : "Approve once"}</button>
      <button disabled={decided || Boolean(pending)} className="danger"
        onClick={() => void decide("reject")}>
        {pending === "reject" ? "Rejecting…" : "Reject"}</button>
    </div>
    <small>No permanent rule is created. This decision applies only to this exact digest.</small>
  </section>;
}
