import { useCallback, useEffect, useRef, useState } from "react";
import { open } from "@tauri-apps/plugin-dialog";
import {
  api,
  ApiError,
  errorMessage,
  type Device,
  type Folder,
  type SharedFolder,
} from "../lib/api";
import { useApi } from "../lib/useApi";
import { isRecent, shortId } from "../lib/format";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusPill } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

interface Snapshot {
  devices: Device[];
  folders: Folder[];
}

async function loadSharing(): Promise<Snapshot> {
  const [devices, folders] = await Promise.all([api.devices(), api.folders()]);
  return { devices, folders };
}

type RemoteState =
  | { state: "loading" }
  | { state: "ready"; folders: SharedFolder[] }
  | { state: "offline" }
  | { state: "error"; message: string };

export default function Transfers() {
  const { data, error, loading, reload } = useApi(loadSharing);
  const [remote, setRemote] = useState<Record<string, RemoteState>>({});
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const fetched = useRef(new Set<string>());

  const friends = (data?.devices ?? []).filter((d) => d.trusted);
  const mine = data?.folders ?? [];
  const mineById = new Map(mine.map((f) => [f.id, f]));

  const fetchFriend = useCallback((id: string) => {
    setRemote((m) => ({ ...m, [id]: { state: "loading" } }));
    api
      .friendFolders(id)
      .then((folders) => {
        setRemote((m) => ({ ...m, [id]: { state: "ready", folders } }));
      })
      .catch((e: unknown) => {
        setRemote((m) => ({
          ...m,
          [id]:
            e instanceof ApiError && e.status === 502
              ? { state: "offline" }
              : { state: "error", message: errorMessage(e) },
        }));
      });
  }, []);

  useEffect(() => {
    for (const f of friends) {
      if (!fetched.current.has(f.id)) {
        fetched.current.add(f.id);
        fetchFriend(f.id);
      }
    }
  }, [friends, fetchFriend]);

  useEffect(() => {
    const t = window.setInterval(reload, 10000);
    return () => window.clearInterval(t);
  }, [reload]);

  async function accept(friend: Device, folder: SharedFolder) {
    const key = `${friend.id}:${folder.id}`;
    setActionError(null);
    try {
      const label = folder.label || folder.id;
      const dest = await open({
        directory: true,
        multiple: false,
        title: `Choose where to keep ${label}`,
      });
      if (typeof dest !== "string") return;
      setBusyKey(key);
      await api.addFolder({ id: folder.id, path: dest, label, direction: "sendreceive" });
      setBusyKey(null);
      reload();
    } catch (e) {
      setBusyKey(null);
      setActionError(errorMessage(e));
    }
  }

  function friendSection(friend: Device) {
    const name = friend.name || shortId(friend.id);
    const state = remote[friend.id] ?? { state: "loading" as const };
    const online = isRecent(friend.last_seen);
    return (
      <section key={friend.id} className="card">
        <header className="card-head">
          <h2 className="card-title">Shared by {name}</h2>
          <div className="row-end">
            {state.state === "ready" && online ? <StatusPill tone="ok" label="Online" /> : null}
            <button
              className="btn btn--ghost btn--sm"
              onClick={() => fetchFriend(friend.id)}
              disabled={state.state === "loading"}
            >
              {state.state === "loading" ? "Checking…" : "Refresh"}
            </button>
          </div>
        </header>
        {state.state === "loading" ? (
          <div className="loading">Asking {name} what they share…</div>
        ) : state.state === "offline" ? (
          <EmptyState
            icon="devices"
            title={`${name} isn't reachable right now`}
            hint="Their shared folders will show up here when their device is back online."
          />
        ) : state.state === "error" ? (
          <Notice message={state.message} onRetry={() => fetchFriend(friend.id)} />
        ) : state.folders.length === 0 ? (
          <EmptyState
            icon="folder"
            title={`${name} isn't sharing any folders yet`}
            hint="Folders they add on their device appear here."
          />
        ) : (
          <ul className="rows">
            {state.folders.map((f) => {
              const local = mineById.get(f.id);
              const key = `${friend.id}:${f.id}`;
              return (
                <li key={f.id} className="row">
                  <span className="row-icon">
                    <Icon name="folder" />
                  </span>
                  <div className="row-body">
                    <p className="row-title">{f.label || f.id}</p>
                    <p className="row-sub mono">{f.id}</p>
                  </div>
                  <div className="row-end">
                    {local ? (
                      <>
                        <span className="row-meta mono" title={local.path}>
                          {local.path}
                        </span>
                        <StatusPill tone="ok" label="Syncing here" />
                      </>
                    ) : (
                      <button
                        className="btn btn--primary btn--sm"
                        onClick={() => accept(friend, f)}
                        disabled={busyKey === key}
                      >
                        {busyKey === key ? "Adding…" : "Add to this PC"}
                      </button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    );
  }

  return (
    <Screen title="Transfers">
      <p className="screen-desc">
        Folders your friends share with you, and what you share with them.
      </p>
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {actionError ? <Notice message={actionError} /> : null}

      {loading && !data ? <div className="loading">Loading…</div> : null}

      {data && friends.length === 0 ? (
        <section className="card">
          <EmptyState
            icon="devices"
            title="No friends yet"
            hint="Add a friend on the Devices screen — their shared folders will show up here."
          />
        </section>
      ) : null}

      {friends.map(friendSection)}

      {data ? (
        <section className="card">
          <header className="card-head">
            <h2 className="card-title">You share</h2>
            <span className="card-count mono">{mine.length}</span>
          </header>
          {mine.length === 0 ? (
            <EmptyState
              icon="folder"
              title="You aren't sharing anything yet"
              hint="Add a folder on the Folders screen and your friends can pick it up here."
            />
          ) : (
            <ul className="rows">
              {mine.map((f) => (
                <li key={f.id} className="row">
                  <span className="row-icon">
                    <Icon name="folder" />
                  </span>
                  <div className="row-body">
                    <p className="row-title">
                      {f.label || f.id}
                      {f.paused ? <span className="pill">Paused</span> : null}
                    </p>
                    <p className="row-sub mono" title={f.path}>
                      {f.path}
                    </p>
                  </div>
                  <div className="row-end">
                    <StatusPill tone="idle" label="Visible to friends" />
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      ) : null}
    </Screen>
  );
}
