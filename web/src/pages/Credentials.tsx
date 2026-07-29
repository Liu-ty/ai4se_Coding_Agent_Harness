import { useCallback, useEffect, useState } from "react";
import type { CredentialStatus } from "../api/types";
import { StatusLabel } from "../components/StatusLabel";

type Operation = "idle" | "loading" | "saving" | "clearing" | "error";

export function Credentials({ loadStatus, onSave, onClear }: {
  loadStatus: (provider: string, host: string) => Promise<CredentialStatus>;
  onSave: (provider: string, host: string, secret: string) => Promise<void>;
  onClear: (provider: string, host: string) => Promise<void>;
}) {
  const [provider, setProvider] = useState("openai");
  const [host, setHost] = useState("api.openai.com");
  const [secret, setSecret] = useState("");
  const [status, setStatus] = useState<CredentialStatus>();
  const [operation, setOperation] = useState<Operation>("idle");
  const [message, setMessage] = useState("");
  const [clearFailed, setClearFailed] = useState(false);

  const refresh = useCallback(async () => {
    setOperation("loading");
    setMessage("Loading credential status…");
    try {
      const next = await loadStatus(provider, host);
      setStatus(next);
      setOperation("idle");
      setMessage(next.configured ? `Configured in ${next.backend}. Secret value is never returned.` :
        "Not configured. Add a credential to this provider and host.");
    } catch {
      setStatus({ ref: { provider, host }, configured: false, backend: "", updated_at: "" });
      setOperation("idle");
      setMessage("Not configured. Secret value is never returned.");
    }
  }, [host, loadStatus, provider]);

  useEffect(() => { void refresh(); }, [refresh]);

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!secret || operation === "saving") return;
    const submittedSecret = secret;
    setSecret("");
    setOperation("saving");
    setMessage(status?.configured ? "Updating credential…" : "Adding credential…");
    void onSave(provider, host, submittedSecret).then(async () => {
      setMessage("Credential saved. Secret value was cleared from this form.");
      await refresh();
    }, () => {
      setOperation("error");
      setMessage("Credential could not be saved. Enter the secret again and retry.");
    });
  };
  const clear = async () => {
    setClearFailed(false);
    setOperation("clearing");
    setMessage("Clearing credential…");
    try {
      await onClear(provider, host);
      setStatus((current) => current ? { ...current, configured: false, backend: "" } : current);
      setOperation("idle");
      setMessage("Credential cleared. No secret value was displayed.");
    } catch {
      setClearFailed(true);
      setOperation("error");
      setMessage("Credential could not be cleared. The existing credential remains unchanged.");
    }
  };

  return <section><header className="page-head"><div><h1>Credentials</h1>
    <p>Status-only storage. Password material is accepted once and never returned.</p></div></header>
    <div className="credential-grid">
      <form className="panel form" onSubmit={submit}>
        <label>Provider<input value={provider} onChange={(event) => setProvider(event.target.value)} /></label>
        <label>Endpoint host<input value={host} onChange={(event) => setHost(event.target.value)} /></label>
        <label>Secret<input type="password" autoComplete="new-password" required value={secret}
          onChange={(event) => setSecret(event.target.value)} /></label>
        <div className="actions">
          <button type="submit" disabled={operation === "saving"}>
            {operation === "saving" ? "Saving…" : status?.configured ? "Update credential" : "Add credential"}
          </button>
          {status?.configured && <button type="button" className="danger"
            disabled={operation === "clearing"} onClick={() => void clear()}>
            {operation === "clearing" ? "Clearing…" : clearFailed ? "Retry clear" : "Clear credential"}
          </button>}
        </div>
      </form>
      <aside className="panel" aria-live="polite"><h2>Credential status</h2>
        {operation === "loading" ? <p>Loading status…</p> :
          <StatusLabel text={status?.configured ? "Configured" : "Not configured"}
            tone={status?.configured ? "success" : operation === "error" ? "danger" : "neutral"} />}
        <p>{message}</p>
        {status?.configured && <dl><dt>Backend</dt><dd>{status.backend}</dd>
          <dt>Updated</dt><dd>{status.updated_at || "Timestamp unavailable"}</dd></dl>}
        <button type="button" disabled={operation === "loading"} onClick={() => void refresh()}>Refresh status</button>
      </aside>
    </div>
  </section>;
}
