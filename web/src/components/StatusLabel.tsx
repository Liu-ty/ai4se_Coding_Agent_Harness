export function StatusLabel({ text, tone = "neutral" }: {
  text: string;
  tone?: "neutral" | "success" | "warning" | "danger";
}) {
  const icon = tone === "success" ? "✓" : tone === "warning" ? "!" : tone === "danger" ? "×" : "◇";
  return <span className={`status status-${tone}`}><span aria-hidden="true">{icon}</span> {text}</span>;
}
