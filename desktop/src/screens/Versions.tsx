import { useState, type FormEvent } from "react";
import { api, errorMessage, type FileVersion } from "../lib/api";
import { useApi } from "../lib/useApi";
import { formatBytes, timeAgo } from "../lib/format";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

export default function Versions() {
  const folders = useApi(api.folders);
  const [folderId, setFolderId] = useState("");
  const [path, setPath] = useState("");
  const [versions, setVersions] = useState<FileVersion[] | null>(null);
  const [searchedPath, setSearchedPath] = useState("");
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);

  const relPath = path.trim();

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!folderId || !relPath || searching) return;
    setSearching(true);
    setSearchError(null);
    api
      .versions(folderId, relPath)
      .then((list) => {
        setSearching(false);
        setVersions([...list].sort((a, b) => b.stamp.localeCompare(a.stamp)));
        setSearchedPath(relPath);
      })
      .catch((err: unknown) => {
        setSearching(false);
        setVersions(null);
        setSearchError(errorMessage(err));
      });
  }

  return (
    <Screen title="Versions">
      <p className="screen-desc">
        Earlier copies of files, kept when changes sync in. Pick a folder, name a file, and see
        what you can recover.
      </p>

      {folders.error ? <Notice message={folders.error} onRetry={folders.reload} /> : null}
      {searchError ? <Notice message={searchError} /> : null}

      <form className="card form-card" onSubmit={submit}>
        <div className="form-grid">
          <label className="field">
            <span className="field-label">Folder</span>
            <select
              className="input select"
              value={folderId}
              onChange={(e) => setFolderId(e.target.value)}
            >
              <option value="">
                {folders.loading && !folders.data ? "Loading folders…" : "Choose a folder"}
              </option>
              {(folders.data ?? []).map((f) => (
                <option key={f.id} value={f.id}>
                  {f.label || f.id}
                </option>
              ))}
            </select>
          </label>
          <label className="field">
            <span className="field-label">File path</span>
            <input
              className="input input--mono"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="docs/report.txt"
            />
          </label>
        </div>
        <div className="form-actions">
          <button
            type="submit"
            className="btn btn--primary"
            disabled={!folderId || !relPath || searching}
          >
            {searching ? "Looking…" : "Show versions"}
          </button>
        </div>
      </form>

      {searching ? <div className="loading">Looking for earlier versions…</div> : null}

      {!searching && versions && versions.length > 0 ? (
        <section className="card">
          <header className="card-head">
            <h2 className="card-title">
              {versions.length === 1 ? "1 earlier version" : `${versions.length} earlier versions`}
              <span className="mono"> · {searchedPath}</span>
            </h2>
          </header>
          <ul className="rows">
            {versions.map((v) => (
              <li key={`${v.stamp}:${v.path}`} className="row">
                <span className="row-icon">
                  <Icon name="versions" />
                </span>
                <div className="row-body">
                  <p className="row-path mono" title={v.path}>
                    {v.stamp}
                  </p>
                  <p className="row-sub">{formatBytes(v.size)}</p>
                </div>
                <div className="row-end">
                  <span className="row-meta">modified {timeAgo(v.mod_time)}</span>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {!searching && versions && versions.length === 0 ? (
        <section className="card">
          <EmptyState
            icon="versions"
            title="No earlier versions"
            hint={`Nothing recovered for "${searchedPath}" yet. A copy is kept each time a synced change overwrites this file.`}
          />
        </section>
      ) : null}
    </Screen>
  );
}
