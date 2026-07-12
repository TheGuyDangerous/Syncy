export type Tone = "ok" | "sync" | "warn" | "danger" | "idle";

export function StatusDot({ tone, pulse = false }: { tone: Tone; pulse?: boolean }) {
  return <span className={`dot dot--${tone}${pulse ? " dot--pulse" : ""}`} aria-hidden="true" />;
}

export function StatusPill({ tone, label }: { tone: Tone; label: string }) {
  return (
    <span className={`status-pill status-pill--${tone}`}>
      <StatusDot tone={tone} />
      {label}
    </span>
  );
}
