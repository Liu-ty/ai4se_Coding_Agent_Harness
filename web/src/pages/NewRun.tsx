import { useState } from "react";
import type { CreateRunRequest, PermissionProfile } from "../api/types";

const initial: CreateRunRequest = {
  repo_root: "", task: "", provider: "", model: "", endpoint: "",
  confirm_custom_endpoint: false, profile: "supervised",
};

export function NewRun({ onCreate }: { onCreate: (request: CreateRunRequest) => void | Promise<void> }) {
  const [draft, setDraft] = useState(initial);
  const field = (name: keyof CreateRunRequest) => ({
    value: String(draft[name] ?? ""),
    onChange: (event: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
      setDraft({ ...draft, [name]: event.target.value }),
  });
  return <section>
    <header className="page-head"><div><h1>New Run</h1><p>Define a bounded run and validate it before execution.</p></div></header>
    <form className="panel form" onSubmit={(event) => { event.preventDefault(); void onCreate(draft); }}>
      <label>Repository path<input required {...field("repo_root")} /></label>
      <label>Task description<textarea required value={draft.task}
        onChange={(event) => setDraft({ ...draft, task: event.target.value })} /></label>
      <label>Provider<input required {...field("provider")} /></label>
      <label>Model<input required {...field("model")} /></label>
      <label>Permission profile<select value={draft.profile}
        onChange={(event) => setDraft({ ...draft, profile: event.target.value as PermissionProfile })}>
        <option value="supervised">Supervised</option><option value="review">Review</option>
        <option value="workspace-auto">Workspace auto</option>
      </select></label>
      <button type="submit">Create supervised draft</button>
      <details><summary>Endpoint options</summary>
        <label>Endpoint<input {...field("endpoint")} placeholder="Provider default" /></label>
      </details>
    </form>
  </section>;
}
