import { useState } from "react";

export function Credentials({ onSave }: {
  onSave: (provider: string, host: string, secret: string) => Promise<void>;
}) {
  const [provider, setProvider] = useState("openai");
  const [host, setHost] = useState("api.openai.com");
  const [secret, setSecret] = useState("");
  const [message, setMessage] = useState("");
  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const submittedSecret = secret;
    setSecret("");
    setMessage("Saving credential…");
    void onSave(provider, host, submittedSecret).then(
      () => setMessage("Credential saved. Secret value was not retained."),
      () => setMessage("Credential was not saved. Check the provider and host."),
    );
  };
  return <section><header className="page-head"><div><h1>Credentials</h1>
    <p>Status-only storage. Existing secret values are never returned.</p></div></header>
    <form className="panel form" onSubmit={submit}>
      <label>Provider<input value={provider} onChange={(event) => setProvider(event.target.value)} /></label>
      <label>Endpoint host<input value={host} onChange={(event) => setHost(event.target.value)} /></label>
      <label>Secret<input type="password" autoComplete="new-password" required value={secret}
        onChange={(event) => setSecret(event.target.value)} /></label>
      <button type="submit">Save credential</button>
      <p role="status">{message}</p>
    </form>
  </section>;
}
