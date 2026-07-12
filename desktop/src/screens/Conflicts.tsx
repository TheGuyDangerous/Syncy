import { api } from "../lib/api";
import { useApi } from "../lib/useApi";
import { Screen } from "../components/Screen";
import { StatusDot } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

export default function Conflicts() {
  const { data, error, loading, reload } = useApi(api.conflicts);

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
            {data.map((c) => (
              <li key={`${c.folder_id}:${c.path}`} className="row">
                <StatusDot tone="warn" />
                <div className="row-body">
                  <p className="row-path mono" title={c.path}>
                    {c.path}
                  </p>
                  <p className="row-sub">
                    in <span className="mono">{c.folder_id}</span>
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </Screen>
  );
}
