export function shortId(id: string): string {
  return id.length > 7 ? id.slice(0, 7) : id;
}

export function timeAgo(iso: string): string {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "unknown";
  const s = Math.max(0, Math.floor((Date.now() - t) / 1000));
  if (s < 45) return "just now";
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d ago`;
  return new Date(t).toLocaleDateString();
}

export function isRecent(iso: string, withinMs = 5 * 60 * 1000): boolean {
  const t = Date.parse(iso);
  return !Number.isNaN(t) && Date.now() - t < withinMs;
}

export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) return "—";
  if (n < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let v = n / 1024;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  return `${v >= 100 ? Math.round(v) : v.toFixed(1)} ${units[i]}`;
}
