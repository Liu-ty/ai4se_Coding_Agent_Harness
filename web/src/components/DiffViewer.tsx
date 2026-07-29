export function DiffViewer({ content, truncated }: { content: string; truncated: boolean }) {
  return <section className="panel">
    <h2>Final diff</h2>
    <pre className="diff" role="region" aria-label="Read-only diff" tabIndex={0}>
      {content.split("\n").map((line, index) =>
        <span key={index} className={line.startsWith("+") ? "add" : line.startsWith("-") ? "del" : ""}>
          {line}{"\n"}
        </span>)}
    </pre>
    {truncated && <p className="notice">Output truncated by the server.</p>}
  </section>;
}
