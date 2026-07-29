import { useEffect, useRef, useState } from "react";
import type {
  CreateRunRequest,
  PermissionProfile,
  PreflightReport,
} from "../api/types";
import { StatusLabel } from "../components/StatusLabel";

const initial: CreateRunRequest = {
  repo_root: "", task: "", provider: "", model: "", endpoint: "",
  confirm_custom_endpoint: false, profile: "supervised",
};

type RequestState = "idle" | "pending" | "success" | "error";

export function NewRun({ onPreflight, onCreate }: {
  onPreflight: (request: CreateRunRequest, signal?: AbortSignal) => Promise<PreflightReport>;
  onCreate: (request: CreateRunRequest) => void | Promise<void>;
}) {
  const [draft, setDraft] = useState(initial);
  const [preflight, setPreflight] = useState<PreflightReport>();
  const [validationState, setValidationState] = useState<RequestState>("idle");
  const [createState, setCreateState] = useState<RequestState>("idle");
  const [message, setMessage] = useState("");
  const [delayed, setDelayed] = useState(false);
  const [validatedFingerprint, setValidatedFingerprint] = useState("");
  const formRef = useRef<HTMLFormElement>(null);
  const generationRef = useRef(0);
  const preflightControllerRef = useRef<AbortController | undefined>(undefined);
  const fingerprint = (value: CreateRunRequest) => JSON.stringify(value);

  useEffect(() => {
    if (validationState !== "pending" && createState !== "pending") {
      setDelayed(false);
      return;
    }
    const timer = window.setTimeout(() => setDelayed(true), 15_000);
    return () => window.clearTimeout(timer);
  }, [validationState, createState]);
  useEffect(() => () => preflightControllerRef.current?.abort(), []);

  const update = <K extends keyof CreateRunRequest>(name: K, value: CreateRunRequest[K]) => {
    generationRef.current += 1;
    preflightControllerRef.current?.abort();
    setDraft((current) => ({ ...current, [name]: value }));
    setPreflight(undefined);
    setValidatedFingerprint("");
    setValidationState("idle");
    setMessage("Draft changed. Validate preflight again.");
  };
  const field = (name: "repo_root" | "provider" | "model" | "endpoint") => ({
    value: draft[name],
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => update(name, event.target.value),
  });
  const validate = async () => {
    if (!formRef.current?.reportValidity()) return;
    preflightControllerRef.current?.abort();
    const controller = new AbortController();
    preflightControllerRef.current = controller;
    const requestFingerprint = fingerprint(draft);
    const generation = ++generationRef.current;
    setValidationState("pending");
    setMessage("Validating repository, configuration, executable, and credential status…");
    try {
      const report = await onPreflight(draft, controller.signal);
      if (controller.signal.aborted || generation !== generationRef.current ||
        requestFingerprint !== fingerprint(draft)) return;
      setPreflight(report);
      setValidatedFingerprint(report.ok ? requestFingerprint : "");
      setValidationState(report.ok ? "success" : "error");
      setMessage(report.ok ? "Preflight passed. Run creation is available." :
        "Preflight did not pass. Resolve the findings and retry.");
    } catch (error) {
      if (controller.signal.aborted || generation !== generationRef.current) return;
      setPreflight(undefined);
      setValidatedFingerprint("");
      setValidationState("error");
      setMessage(error instanceof Error ? error.message : "Preflight request failed. Retry validation.");
    }
  };
  const create = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!preflight?.ok || validatedFingerprint !== fingerprint(draft) ||
      createState === "pending") return;
    setCreateState("pending");
    setMessage("Creating bounded run…");
    try {
      await onCreate(draft);
      setCreateState("success");
    } catch (error) {
      setCreateState("error");
      setMessage(error instanceof Error ? error.message : "Run creation failed. Retry without losing the draft.");
    }
  };

  return <section>
    <header className="page-head"><div><h1>New Run</h1>
      <p>Validate a bounded local request before the server may create it.</p></div></header>
    <div className="form-layout">
      <form ref={formRef} className="panel form" onSubmit={create}>
        <label>Repository path<input required {...field("repo_root")} /></label>
        <label>Task description<textarea required value={draft.task}
          onChange={(event) => update("task", event.target.value)} /></label>
        <label>Provider<input required {...field("provider")} /></label>
        <label>Model<input required {...field("model")} /></label>
        <label>Permission profile<select value={draft.profile}
          onChange={(event) => update("profile", event.target.value as PermissionProfile)}>
          <option value="supervised">Supervised</option><option value="review">Review</option>
          <option value="workspace-auto">Workspace auto</option>
        </select></label>
        <div className="actions">
          <button type="button" disabled={validationState === "pending"} onClick={() => void validate()}>
            {validationState === "pending" ? "Validating…" : validationState === "error" ? "Retry preflight" : "Validate preflight"}
          </button>
          <button type="submit" disabled={!preflight?.ok ||
            validatedFingerprint !== fingerprint(draft) || createState === "pending"}>
            {createState === "pending" ? "Creating…" : createState === "error" ? "Retry create" : "Create run"}
          </button>
        </div>
        <details><summary>Endpoint options</summary>
          <label>Endpoint<input {...field("endpoint")} placeholder="Provider default" /></label>
        </details>
      </form>
      <aside className="panel preflight-rail" aria-live="polite">
        <h2>Preflight</h2>
        <StatusLabel text={validationState === "success" ? "Passed" :
          validationState === "pending" ? "Checking" :
          validationState === "error" ? "Action required" : "Not validated"}
          tone={validationState === "success" ? "success" :
            validationState === "error" ? "danger" : validationState === "pending" ? "warning" : "neutral"} />
        <p>{message || "Complete the form, then validate it against the local server."}</p>
        {delayed && <p className="notice">Taking longer than expected. You may continue waiting or retry after the request finishes.</p>}
        {preflight && <ul className="finding-list">
          {preflight.findings.length === 0 && <li>No findings.</li>}
          {preflight.findings.map((finding) => <li key={`${finding.code}-${finding.message}`}>
            <StatusLabel text={finding.code} tone={finding.severity === "error" ? "danger" :
              finding.severity === "warning" ? "warning" : "success"} /> <span>{finding.message}</span>
          </li>)}
        </ul>}
      </aside>
    </div>
  </section>;
}
