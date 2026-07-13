import { useEffect, useState } from "react";
import { api } from "./lib/api";
import { useApi } from "./lib/useApi";
import { initTheme } from "./lib/theme";
import { shortId } from "./lib/format";
import { BrandMark } from "./components/BrandMark";
import { Icon, type IconName } from "./components/Icon";
import { StatusDot, type Tone } from "./components/StatusDot";
import { TitleBar } from "./components/TitleBar";
import Dashboard from "./screens/Dashboard";
import Devices from "./screens/Devices";
import Folders from "./screens/Folders";
import Conflicts from "./screens/Conflicts";
import Placeholder from "./screens/Placeholder";
import Settings from "./screens/Settings";
import Versions from "./screens/Versions";

initTheme();

type View =
  | "dashboard"
  | "devices"
  | "folders"
  | "transfers"
  | "history"
  | "versions"
  | "conflicts"
  | "logs"
  | "settings";

const NAV: { view: View; label: string; icon: IconName }[] = [
  { view: "dashboard", label: "Dashboard", icon: "dashboard" },
  { view: "devices", label: "Devices", icon: "devices" },
  { view: "folders", label: "Folders", icon: "folder" },
  { view: "transfers", label: "Transfers", icon: "transfers" },
  { view: "history", label: "History", icon: "history" },
  { view: "versions", label: "Versions", icon: "versions" },
  { view: "conflicts", label: "Conflicts", icon: "conflicts" },
  { view: "logs", label: "Logs", icon: "logs" },
  { view: "settings", label: "Settings", icon: "settings" },
];

async function loadShell() {
  const [status, conflicts] = await Promise.all([api.status(), api.conflicts()]);
  return { status, conflictCount: conflicts.length };
}

function screenFor(view: View) {
  switch (view) {
    case "dashboard":
      return <Dashboard />;
    case "devices":
      return <Devices />;
    case "folders":
      return <Folders />;
    case "transfers":
      return (
        <Placeholder
          title="Transfers"
          description="File activity between this device and your peers."
          icon="transfers"
          emptyTitle="Nothing moving right now"
          emptyHint="Active uploads and downloads appear here as they happen."
        />
      );
    case "history":
      return (
        <Placeholder
          title="History"
          description="A record of what synced, when, and from where."
          icon="history"
          emptyTitle="No activity yet"
          emptyHint="Changes show up here as your folders sync."
        />
      );
    case "versions":
      return <Versions />;
    case "conflicts":
      return <Conflicts />;
    case "logs":
      return (
        <Placeholder
          title="Logs"
          description="Raw engine output for troubleshooting."
          icon="logs"
          emptyTitle="No log entries yet"
          emptyHint="Engine events stream here while Syncy runs."
        />
      );
    case "settings":
      return <Settings />;
  }
}

export default function App() {
  const [view, setView] = useState<View>("dashboard");
  const shell = useApi(loadShell);

  useEffect(() => {
    const t = window.setInterval(shell.reload, 15000);
    return () => window.clearInterval(t);
  }, [shell.reload]);

  useEffect(() => {
    if (shell.data || shell.loading) return;
    const t = window.setTimeout(shell.reload, 1500);
    return () => window.clearTimeout(t);
  }, [shell.data, shell.loading, shell.error, shell.reload]);

  let tone: Tone;
  let label: string;
  if (shell.data) {
    if (shell.data.conflictCount > 0) {
      tone = "warn";
      label =
        shell.data.conflictCount === 1 ? "1 conflict" : `${shell.data.conflictCount} conflicts`;
    } else {
      tone = "ok";
      label = "All synced";
    }
  } else if (shell.loading) {
    tone = "idle";
    label = "Connecting…";
  } else {
    tone = "idle";
    label = "Engine offline";
  }

  return (
    <div className="app">
      <TitleBar />
      <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <BrandMark size={30} />
          <span className="wordmark">Syncy</span>
        </div>
        <nav className="nav" aria-label="Main">
          {NAV.map((item) => (
            <button
              key={item.view}
              className="nav-item"
              aria-current={view === item.view ? "page" : undefined}
              onClick={() => setView(item.view)}
            >
              <span className="nav-ic">
                <Icon name={item.icon} />
              </span>
              <span className="nav-label">{item.label}</span>
            </button>
          ))}
        </nav>
        <div className="side-foot">
          <StatusDot tone={tone} />
          <div className="side-foot-text">
            <span className="side-foot-label">{label}</span>
            <span
              className="mono side-foot-id"
              title={shell.data ? shell.data.status.device_id : undefined}
            >
              {shell.data ? shortId(shell.data.status.device_id) : "—"}
            </span>
          </div>
        </div>
      </aside>
      <main className="main">{screenFor(view)}</main>
      </div>
    </div>
  );
}
