import { api, type Conflict, type Device, type Folder, type Status } from "../lib/api";
import { useApi } from "../lib/useApi";
import { isRecent, isZeroTime, shortId, timeAgo } from "../lib/format";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusDot, StatusPill } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

interface Snapshot {
  status: Status;
  folders: Folder[];
  devices: Device[];
  conflicts: Conflict[];
}

async function loadSnapshot(): Promise<Snapshot> {
  const [status, folders, devices, conflicts] = await Promise.all([
    api.status(),
    api.folders(),
    api.devices(),
    api.conflicts(),
  ]);
  return { status, folders, devices, conflicts };
}

export default function Dashboard() {
  const { data, error, loading, reload } = useApi(loadSnapshot);

  return (
    <Screen title="Dashboard">
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {loading && !data ? <div className="loading">Loading…</div> : null}
      {data ? <Overview snapshot={data} /> : null}
    </Screen>
  );
}

function Overview({ snapshot }: { snapshot: Snapshot }) {
  const { status, folders, devices, conflicts } = snapshot;
  const conflicted = conflicts.length;
  const online = devices.filter((d) => isRecent(d.last_seen)).length;
  const paused = folders.filter((f) => f.paused).length;

  return (
    <>
      <section className="card hero-card">
        <div className="hero-state">
          <StatusDot tone={conflicted > 0 ? "warn" : "ok"} />
          <div>
            <p className="hero-title">{conflicted > 0 ? "Attention needed" : "All synced"}</p>
            <p className="hero-sub">
              {conflicted > 0
                ? `${conflicted} ${conflicted === 1 ? "file has" : "files have"} conflicting copies.`
                : "Everything is up to date."}
            </p>
          </div>
        </div>
        <div className="hero-id">
          <span className="hero-id-label">This device</span>
          <span className="mono" title={status.device_id}>
            {shortId(status.device_id)}
          </span>
        </div>
      </section>

      <section className="tiles">
        <div className="card tile">
          <div className="tile-top">
            <span>Folders</span>
            <Icon name="folder" />
          </div>
          <p className="tile-value">{folders.length}</p>
          <div>
            {folders.length === 0 ? (
              <StatusPill tone="idle" label="None yet" />
            ) : paused > 0 ? (
              <StatusPill tone="idle" label={`${paused} paused`} />
            ) : (
              <StatusPill tone="ok" label="All active" />
            )}
          </div>
        </div>
        <div className="card tile">
          <div className="tile-top">
            <span>Devices</span>
            <Icon name="devices" />
          </div>
          <p className="tile-value">{devices.length}</p>
          <div>
            {devices.length === 0 ? (
              <StatusPill tone="idle" label="None paired" />
            ) : online > 0 ? (
              <StatusPill tone="ok" label={`${online} online`} />
            ) : (
              <StatusPill tone="idle" label="All offline" />
            )}
          </div>
        </div>
        <div className="card tile">
          <div className="tile-top">
            <span>Conflicts</span>
            <Icon name="conflicts" />
          </div>
          <p className="tile-value">{conflicted}</p>
          <div>
            {conflicted > 0 ? (
              <StatusPill tone="warn" label="Needs review" />
            ) : (
              <StatusPill tone="ok" label="All clear" />
            )}
          </div>
        </div>
      </section>

      <section className="card">
        <header className="card-head">
          <h2 className="card-title">Connected devices</h2>
          <span className="card-count mono">{devices.length}</span>
        </header>
        {devices.length === 0 ? (
          <EmptyState
            icon="devices"
            title="No devices yet"
            hint="Pair another device to start syncing with it."
          />
        ) : (
          <ul className="rows">
            {devices.map((d) => {
              const active = isRecent(d.last_seen);
              return (
                <li key={d.id} className="row">
                  <StatusDot tone={active ? "ok" : "idle"} />
                  <div className="row-body">
                    <p className="row-title">
                      {d.name}
                      {d.trusted ? (
                        <span className="pill pill--trusted">
                          <Icon name="shield" size={11} />
                          Trusted
                        </span>
                      ) : null}
                    </p>
                    <p className="row-sub mono" title={d.id}>
                      {shortId(d.id)}
                    </p>
                  </div>
                  <span className="row-meta">
                    {active
                      ? "Online"
                      : isZeroTime(d.last_seen)
                        ? "Not seen yet"
                        : `Seen ${timeAgo(d.last_seen)}`}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </>
  );
}
