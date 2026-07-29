export function DemoGallery({ fixedRuns, onOpen }: {
  fixedRuns: string[];
  onOpen: (runId: string) => void;
}) {
  return <section>
    <div className="sim-banner">◇ <strong>SIMULATED</strong> DEMO ENVIRONMENT — no real provider, credential, filesystem, or process execution.</div>
    <header className="page-head"><div><h1>Demo Gallery</h1>
      <p>Fixed scenarios demonstrate the feedback loop without production capabilities.</p></div></header>
    <div className="cards">{fixedRuns.map((runId) => <article className="panel demo-card" key={runId}>
      <strong className="sim-label">SIMULATED</strong>
      <h2>Feedback-loop repair</h2>
      <p>Policy interception, injected validation failure, changed second patch, and objective success.</p>
      <button onClick={() => onOpen(runId)}>Open SIMULATED demo</button>
    </article>)}</div>
    <aside className="panel"><h2>SIMULATED capability boundary</h2>
      <p>Arbitrary run creation, credentials, configuration validation, approval mutations, cancellation,
        and non-fixed artifacts are unavailable.</p></aside>
  </section>;
}
