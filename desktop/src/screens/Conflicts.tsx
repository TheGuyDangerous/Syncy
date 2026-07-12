import { useState } from "react";
import { api, errorMessage } from "../lib/api";
import { useApi } from "../lib/useApi";
import { Screen } from "../components/Screen";
import { StatusDot } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";
import { Icon } from "../components/Icon";

interface Note {
  ok: boolean;
  text: string;
}

export default function Conflicts() {
  const { data, error, loading, reload } = useApi(api.conflicts);
  const [busy, setBusy] = useState<string | null>(null);
  const [notes, setNotes] = useState<Record<string, Note>>({});

  async function explain(folderId: string, path: string) {
    const key = `${folderId}:${path}`;
    setBusy(key);
    try {
      const res = await api.explainConflict({ folder: folderId, path });
      setNotes((n) => ({ ...n, [key]: { ok: true, text: res.text } }));
    } catch (err) {
      const msg = errorMessage(err);
      setNotes((n) => ({
        ...n,
        [key]: {
          ok: false,
          text: msg.includes("disabled")
            ? "Turn on the AI assistant in Settings → Integrations to use this."
            : msg,
        },
      }));
    } finally {
      setBusy(null);
    }
  }

  return (
    <Screen title="Conflicts">
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {loading && !data ? <div className="loading">Loading…</div> : null}

      {data && data.length === 0 ? (
        <section className="card">
          <EmptyState
            icon="conflicts"
            title="No conflicts"
            hint="When two devices change the same file before it syncs, both copies land here for you to pick from."
          />
        </section>
      ) : null}

      {data && data.length > 0 ? (
        <section className="card">
          <header className="card-head">
            <h2 className="card-title">
              {data.length === 1 ? "1 conflicting file" : `${data.length} conflicting files`}
            </h2>
          </header>
          <ul className="rows">
            {data.map((c) => {
              const key = `${c.folder_id}:${c.path}`;
              const note = notes[key];
              return (
                <li key={key} className="row row--stack">
                  <div className="row-main">
                    <StatusDot tone="warn" />
                    <div className="row-body">
                      <p className="row-path mono" title={c.path}>
                        {c.path}
                      </p>
                      <p className="row-sub">
                        in <span className="mono">{c.folder_id}</span>
                      </p>
                    </div>
                    <div className="row-end">
                      <button
                        className="btn btn--sm"
                        onClick={() => explain(c.folder_id, c.path)}
                        disabled={busy === key}
                      >
                        <Icon name="sparkle" size={13} />
                        {busy === key ? "Explaining…" : "Explain"}
                      </button>
                    </div>
                  </div>
                  {note ? (
                    <p className={note.ok ? "conflict-note" : "form-error"} role="status">
                      {note.text}
                    </p>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
    </Screen>
  );
}
