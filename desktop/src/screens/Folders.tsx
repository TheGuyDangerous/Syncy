import { useState, type FormEvent } from "react";
import { api, errorMessage, type Direction } from "../lib/api";
import { useApi } from "../lib/useApi";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusPill } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

const DIRECTION_LABEL: Record<Direction, string> = {
  sendreceive: "Send & receive",
  sendonly: "Send only",
  receiveonly: "Receive only",
};

export default function Folders() {
  const { data, error, loading, reload } = useApi(api.folders);
  const [showAdd, setShowAdd] = useState(false);
  const [id, setId] = useState("");
  const [path, setPath] = useState("");
  const [label, setLabel] = useState("");
  const [direction, setDirection] = useState<Direction>("sendreceive");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  function openForm() {
    setShowAdd(true);
    setFormError(null);
  }

  function closeForm() {
    setShowAdd(false);
    setFormError(null);
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    const folderId = id.trim();
    const folderPath = path.trim();
    if (!folderId || !folderPath) {
      setFormError("Folder ID and path are required.");
      return;
    }
    setSubmitting(true);
    setFormError(null);
    api
      .addFolder({
        id: folderId,
        path: folderPath,
        label: label.trim() || undefined,
        direction,
      })
      .then(() => {
        setSubmitting(false);
        setShowAdd(false);
        setId("");
        setPath("");
        setLabel("");
        setDirection("sendreceive");
        reload();
      })
      .catch((err: unknown) => {
        setSubmitting(false);
        setFormError(errorMessage(err));
      });
  }

  function remove(folderId: string) {
    setBusyId(folderId);
    setActionError(null);
    api
      .removeFolder(folderId)
      .then(() => {
        setBusyId(null);
        setConfirmId(null);
        reload();
      })
      .catch((err: unknown) => {
        setBusyId(null);
        setConfirmId(null);
        setActionError(errorMessage(err));
      });
  }

  return (
    <Screen
      title="Folders"
      actions={
        <button className="btn btn--primary" onClick={() => (showAdd ? closeForm() : openForm())}>
          <Icon name="plus" size={14} />
          Add folder
        </button>
      }
    >
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {actionError ? <Notice message={actionError} /> : null}

      {showAdd ? (
        <form className="card form-card" onSubmit={submit}>
          <div className="form-grid">
            <label className="field">
              <span className="field-label">Folder ID</span>
              <input
                className="input input--mono"
                value={id}
                onChange={(e) => setId(e.target.value)}
                placeholder="documents"
              />
            </label>
            <label className="field">
              <span className="field-label">Path</span>
              <input
                className="input input--mono"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="C:\Users\you\Documents"
              />
            </label>
            <label className="field">
              <span className="field-label">Label (optional)</span>
              <input
                className="input"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="Documents"
              />
            </label>
            <label className="field">
              <span className="field-label">Direction</span>
              <select
                className="input select"
                value={direction}
                onChange={(e) => setDirection(e.target.value as Direction)}
              >
                <option value="sendreceive">Send & receive</option>
                <option value="sendonly">Send only</option>
                <option value="receiveonly">Receive only</option>
              </select>
            </label>
          </div>
          {formError ? (
            <p className="form-error" role="alert">
              {formError}
            </p>
          ) : null}
          <div className="form-actions">
            <button type="submit" className="btn btn--primary" disabled={submitting}>
              {submitting ? "Adding…" : "Add folder"}
            </button>
            <button type="button" className="btn btn--ghost" onClick={closeForm}>
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {loading && !data ? <div className="loading">Loading…</div> : null}

      {data && data.length > 0 ? (
        <section className="card">
          <ul className="rows">
            {data.map((f) => (
              <li key={f.id} className="row">
                <span className="row-icon">
                  <Icon name="folder" />
                </span>
                <div className="row-body">
                  <p className="row-title">
                    {f.label || f.id}
                    <span className="pill">{DIRECTION_LABEL[f.direction]}</span>
                  </p>
                  <p className="row-sub mono" title={f.path}>
                    {f.path}
                  </p>
                </div>
                <div className="row-end">
                  {confirmId === f.id ? (
                    <>
                      <span className="row-meta">Stop syncing? Files stay on disk.</span>
                      <button
                        className="btn btn--danger btn--sm"
                        onClick={() => remove(f.id)}
                        disabled={busyId === f.id}
                      >
                        {busyId === f.id ? "Removing…" : "Remove"}
                      </button>
                      <button className="btn btn--ghost btn--sm" onClick={() => setConfirmId(null)}>
                        Cancel
                      </button>
                    </>
                  ) : (
                    <>
                      {f.paused ? (
                        <StatusPill tone="idle" label="Paused" />
                      ) : (
                        <StatusPill tone="ok" label="Synced" />
                      )}
                      <button
                        className="icon-btn icon-btn--danger"
                        aria-label={`Remove ${f.label || f.id}`}
                        onClick={() => setConfirmId(f.id)}
                      >
                        <Icon name="trash" size={15} />
                      </button>
                    </>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {data && data.length === 0 ? (
        <section className="card">
          <EmptyState
            icon="folder"
            title="No folders yet"
            hint="Add a folder to start syncing it across your devices."
            action={
              !showAdd ? (
                <button className="btn" onClick={openForm}>
                  <Icon name="plus" size={14} />
                  Add folder
                </button>
              ) : undefined
            }
          />
        </section>
      ) : null}
    </Screen>
  );
}
