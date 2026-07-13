import { useEffect, useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";

const appWindow = getCurrentWindow();

type Dir =
  | "North"
  | "South"
  | "East"
  | "West"
  | "NorthEast"
  | "NorthWest"
  | "SouthEast"
  | "SouthWest";

const HANDLES: { dir: Dir; cls: string }[] = [
  { dir: "North", cls: "n" },
  { dir: "South", cls: "s" },
  { dir: "East", cls: "e" },
  { dir: "West", cls: "w" },
  { dir: "NorthWest", cls: "nw" },
  { dir: "SouthEast", cls: "se" },
  { dir: "SouthWest", cls: "sw" },
];

export function TitleBar() {
  const [maximized, setMaximized] = useState(false);

  useEffect(() => {
    let unlisten: (() => void) | undefined;
    appWindow.isMaximized().then(setMaximized).catch(() => {});
    appWindow
      .onResized(() => {
        appWindow.isMaximized().then(setMaximized).catch(() => {});
      })
      .then((u) => {
        unlisten = u;
      })
      .catch(() => {});
    return () => unlisten?.();
  }, []);

  return (
    <>
      {!maximized
        ? HANDLES.map((h) => (
            <div
              key={h.cls}
              className={`resize-handle resize-handle--${h.cls}`}
              onMouseDown={(e) => {
                if (e.button === 0) void appWindow.startResizeDragging(h.dir);
              }}
            />
          ))
        : null}

      <header className="titlebar" data-tauri-drag-region>
        <div className="titlebar-brand" data-tauri-drag-region>
          <span className="titlebar-mark">S</span>
          <span className="titlebar-name">Syncy</span>
        </div>
        <div className="titlebar-controls">
          <button
            className="win-btn"
            onClick={() => void appWindow.minimize()}
            aria-label="Minimize"
            title="Minimize"
          >
            <svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
              <path d="M2.5 6h7" stroke="currentColor" strokeWidth="1.1" />
            </svg>
          </button>
          <button
            className="win-btn"
            onClick={() => void appWindow.toggleMaximize()}
            aria-label={maximized ? "Restore" : "Maximize"}
            title={maximized ? "Restore" : "Maximize"}
          >
            {maximized ? (
              <svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
                <rect x="3.5" y="1.5" width="6.5" height="6.5" fill="none" stroke="currentColor" strokeWidth="1.1" />
                <rect x="1.5" y="3.5" width="6.5" height="6.5" fill="var(--bg)" stroke="currentColor" strokeWidth="1.1" />
              </svg>
            ) : (
              <svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
                <rect x="2" y="2" width="8" height="8" fill="none" stroke="currentColor" strokeWidth="1.1" />
              </svg>
            )}
          </button>
          <button
            className="win-btn win-btn--close"
            onClick={() => void appWindow.close()}
            aria-label="Close"
            title="Close"
          >
            <svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
              <path d="M2.6 2.6l6.8 6.8M9.4 2.6l-6.8 6.8" stroke="currentColor" strokeWidth="1.1" />
            </svg>
          </button>
        </div>
      </header>
    </>
  );
}
