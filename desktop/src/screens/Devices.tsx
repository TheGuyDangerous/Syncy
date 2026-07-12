import { api } from "../lib/api";
import { useApi } from "../lib/useApi";
import { isRecent, shortId, timeAgo } from "../lib/format";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusDot } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

export default function Devices() {
  const { data, error, loading, reload } = useApi(api.devices);

  return (
    <Screen title="Devices">
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {loading && !data ? <div className="loading">Loading…</div> : null}

      {data && data.length === 0 ? (
        <section className="card">
          <EmptyState
            icon="devices"
            title="No devices yet"
            hint="Pair another device to sync folders with it."
          />
        </section>
      ) : null}

      {data && data.length > 0 ? (
        <section className="card">
          <ul className="rows">
            {data.map((d) => {
              const active = isRecent(d.last_seen);
              return (
                <li key={d.id} className="row">
                  <span className="row-icon">
                    <Icon name="devices" />
                  </span>
                  <div className="row-body">
                    <p className="row-title">
                      {d.name}
                      {d.trusted ? (
                        <span className="pill pill--trusted">
                          <Icon name="shield" size={11} />
                          Trusted
                        </span>
                      ) : (
                        <span className="pill">Not trusted</span>
                      )}
                    </p>
                    <p className="row-sub mono" title={d.id}>
                      {shortId(d.id)}
                    </p>
                  </div>
                  <div className="row-end">
                    <span className="row-meta">
                      {active ? "Online" : `Seen ${timeAgo(d.last_seen)}`}
                    </span>
                    <StatusDot tone={active ? "ok" : "idle"} />
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
    </Screen>
  );
}
