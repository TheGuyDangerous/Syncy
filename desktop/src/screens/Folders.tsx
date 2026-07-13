import { useMemo, useState, type FormEvent } from "react";
import { open } from "@tauri-apps/plugin-dialog";
import { api, errorMessage, type Direction } from "../lib/api";
import { useApi } from "../lib/useApi";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { Select } from "../components/Select";
import { StatusPill } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

const DIRECTION_LABEL: Record<Direction, string> = {
  sendreceive: "Send & receive",
  sendonly: "Send only",
  receiveonly: "Receive only",
};

const DIRECTION_OPTIONS = Object.entries(DIRECTION_LABEL).map(([value, label]) => ({
  value,
  label,
}));

function baseName(p: string) {
  return p.split(/[\\/]/).filter(Boolean).pop() ?? p;
}

function slugId(name: string, taken: Set<string>) {
  const base =
    name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "folder";
  if (!taken.has(base)) return base;
  let n = 2;
  while (taken.has(`${base}-${n}`)) n++;
  return `${base}-${n}`;
}

export default function Folders() {
  const { data, error, loading, reload } = useApi(api.folders);

  const [pickedPath, setPickedPath] = useState<string | null>(null);
  const [pickName, setPickName] = useState("");
  const [pickDirection, setPickDirection] = useState<Direction>("sendreceive");
  const [pickBusy, setPickBusy] = useState(false);
  const [pickError, setPickError] = useState<string | null>(null);

  const [id, setId] = useState("");
  const [path, setPath] = useState("");
  const [label, setLabel] = useState("");
  const [direction, setDirection] = useState<Direction>("sendreceive");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const pickedId = useMemo(() => {
    if (!pickedPath) return "";
    return slugId(baseName(pickedPath), new Set((data ?? []).map((f) => f.id)));
  }, [pickedPath, data]);

  async function choose() {
    setPickError(null);
    try {
      const dir = await open({ directory: true, multiple: false, title: "Choose a folder to sync" });
      if (typeof dir !== "string") return;
      setPickedPath(dir);
      setPickName(baseName(dir));
      setPickDirection("sendreceive");
    } catch (err) {
      setPickError(errorMessage(err));
    }
  }

  function resetPicker() {
    setPickedPath(null);
    setPickName("");
    setPickDirection("sendreceive");
    setPickError(null);
  }

  function addPicked(e: FormEvent) {
    e.preventDefault();
    if (!pickedPath || pickBusy) return;
    setPickBusy(true);
    setPickError(null);
    api
      .addFolder({
        id: pickedId,
        path: pickedPath,
        label: pickName.trim() || undefined,
        direction: pickDirection,
      })
      .then(() => {
        setPickBusy(false);
        resetPicker();
        reload();
      })
      .catch((err: unknown) => {
        setPickBusy(false);
        setPickError(errorMessage(err));
      });
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
        <button className="btn btn--primary" onClick={choose}>
          <Icon name="plus" size={14} />
          Add folder
        </button>
      }
    >
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {actionError ? <Notice message={actionError} /> : null}

      {pickedPath ? (
        <form className="card form-card" onSubmit={addPicked}>
          <div className="field">
            <span className="field-label">Folder</span>
            <p className="picker-path" title={pickedPath}>
              {pickedPath}
            </p>
          </div>
          <div className="form-grid">
            <label className="field">
              <span className="field-label">Name</span>
              <input
                className="input"
                value={pickName}
                onChange={(e) => setPickName(e.target.value)}
                placeholder={baseName(pickedPath)}
              />
            </label>
            <div className="field">
              <span className="field-label">Direction</span>
              <Select
                value={pickDirection}
                onChange={(v) => setPickDirection(v as Direction)}
                options={DIRECTION_OPTIONS}
                ariaLabel="Direction"
              />
            </div>
          </div>
          <p className="setting-hint">
            Adds as <span className="mono">{pickedId}</span>
          </p>
          {pickError ? (
            <p className="form-error" role="alert">
              {pickError}
            </p>
          ) : null}
          <div className="form-actions">
            <button type="submit" className="btn btn--primary" disabled={pickBusy}>
              {pickBusy ? "Adding…" : "Add folder"}
            </button>
            <button type="button" className="btn btn--ghost" onClick={choose}>
              Choose another…
            </button>
            <button type="button" className="btn btn--ghost" onClick={resetPicker}>
              Cancel
            </button>
          </div>
        </form>
      ) : (
        <section className="card">
          <div className="picker-row">
            <span className="row-icon">
              <Icon name="folder" />
            </span>
            <div className="row-body">
              <p className="row-title">Sync a new folder</p>
              <p className="row-sub">
                Pick a folder on this device and Syncy keeps it in step everywhere.
              </p>
              {pickError ? (
                <p className="form-error" role="alert">
                  {pickError}
                </p>
              ) : null}
            </div>
            <button className="btn btn--primary" onClick={choose}>
              Choose a folder…
            </button>
          </div>
        </section>
      )}

      <details className="card adv-add">
        <summary>
          <svg
            className="adv-chevron"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <path d="m9 6 6 6-6 6" />
          </svg>
          Advanced — add by path
        </summary>
        <form className="form-card" onSubmit={submit}>
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
            <div className="field">
              <span className="field-label">Direction</span>
              <Select
                value={direction}
                onChange={(v) => setDirection(v as Direction)}
                options={DIRECTION_OPTIONS}
                ariaLabel="Direction"
              />
            </div>
          </div>
          {formError ? (
            <p className="form-error" role="alert">
              {formError}
            </p>
          ) : null}
          <div className="form-actions">
            <button type="submit" className="btn" disabled={submitting}>
              {submitting ? "Adding…" : "Add folder"}
            </button>
          </div>
        </form>
      </details>

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
              <button className="btn" onClick={choose}>
                <Icon name="plus" size={14} />
                Choose a folder…
              </button>
            }
          />
        </section>
      ) : null}
    </Screen>
  );
}
