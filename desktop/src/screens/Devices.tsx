import { useRef, useState, type FormEvent } from "react";
import { api, errorMessage, type Device, type Status } from "../lib/api";
import { useApi } from "../lib/useApi";
import { isRecent, isZeroTime, shortId, timeAgo } from "../lib/format";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusDot } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

interface Snapshot {
  status: Status;
  devices: Device[];
}

async function loadDevices(): Promise<Snapshot> {
  const [status, devices] = await Promise.all([api.status(), api.devices()]);
  return { status, devices };
}

export default function Devices() {
  const { data, error, loading, reload } = useApi(loadDevices);
  const [showPair, setShowPair] = useState(false);
  const [pairId, setPairId] = useState("");
  const [pairName, setPairName] = useState("");
  const [pairBusy, setPairBusy] = useState(false);
  const [pairError, setPairError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef<number | undefined>(undefined);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  function openPair() {
    setShowPair(true);
    setPairError(null);
  }

  function closePair() {
    setShowPair(false);
    setPairError(null);
  }

  async function copyOwnId() {
    if (!data) return;
    const id = data.status.device_id;
    try {
      await navigator.clipboard.writeText(id);
    } catch {
      const ta = document.createElement("textarea");
      ta.value = id;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      ta.remove();
    }
    setCopied(true);
    window.clearTimeout(copyTimer.current);
    copyTimer.current = window.setTimeout(() => setCopied(false), 1500);
  }

  function pair(e: FormEvent) {
    e.preventDefault();
    const id = pairId.trim();
    if (!id || pairBusy) return;
    setPairBusy(true);
    setPairError(null);
    api
      .pairDevice({ id, name: pairName.trim() || undefined })
      .then(() => {
        setPairBusy(false);
        setShowPair(false);
        setPairId("");
        setPairName("");
        reload();
      })
      .catch((err: unknown) => {
        setPairBusy(false);
        setPairError(errorMessage(err));
      });
  }

  function unpair(deviceId: string) {
    setBusyId(deviceId);
    setActionError(null);
    api
      .unpairDevice(deviceId)
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
      title="Devices"
      actions={
        <button className="btn btn--primary" onClick={() => (showPair ? closePair() : openPair())}>
          <Icon name="plus" size={14} />
          Pair a device
        </button>
      }
    >
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {actionError ? <Notice message={actionError} /> : null}

      {data ? (
        <section className="card">
          <div className="setting-row">
            <div>
              <p className="setting-name">This device</p>
              <p className="setting-hint">Share this id with your other device to pair.</p>
              <p className="mono device-id" title={data.status.device_id}>
                {data.status.device_id}
              </p>
            </div>
            <button className="btn btn--sm" onClick={copyOwnId}>
              <Icon name="copy" size={13} />
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
        </section>
      ) : null}

      {showPair ? (
        <form className="card form-card" onSubmit={pair}>
          <div className="form-grid">
            <label className="field">
              <span className="field-label">Device ID</span>
              <input
                className="input input--mono"
                value={pairId}
                onChange={(e) => setPairId(e.target.value)}
                placeholder="paste the other device's id"
              />
            </label>
            <label className="field">
              <span className="field-label">Name (optional)</span>
              <input
                className="input"
                value={pairName}
                onChange={(e) => setPairName(e.target.value)}
                placeholder="My laptop"
              />
            </label>
          </div>
          <p className="setting-hint">
            Pair on both devices — add each other's id — then they'll find each other on your
            network.
          </p>
          {pairError ? (
            <p className="form-error" role="alert">
              {pairError}
            </p>
          ) : null}
          <div className="form-actions">
            <button
              type="submit"
              className="btn btn--primary"
              disabled={pairBusy || !pairId.trim()}
            >
              {pairBusy ? "Pairing…" : "Pair device"}
            </button>
            <button type="button" className="btn btn--ghost" onClick={closePair}>
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {loading && !data ? <div className="loading">Loading…</div> : null}

      {data && data.devices.length === 0 ? (
        <section className="card">
          <EmptyState
            icon="devices"
            title="No devices yet"
            hint="Pair one to start syncing — add each other's id on both devices."
            action={
              !showPair ? (
                <button className="btn" onClick={openPair}>
                  <Icon name="plus" size={14} />
                  Pair a device
                </button>
              ) : undefined
            }
          />
        </section>
      ) : null}

      {data && data.devices.length > 0 ? (
        <section className="card">
          <ul className="rows">
            {data.devices.map((d) => {
              const active = isRecent(d.last_seen);
              return (
                <li key={d.id} className="row">
                  <span className="row-icon">
                    <Icon name="devices" />
                  </span>
                  <div className="row-body">
                    <p className="row-title">
                      {d.name || shortId(d.id)}
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
                    {confirmId === d.id ? (
                      <>
                        <span className="row-meta">Unpair? Syncing with it stops.</span>
                        <button
                          className="btn btn--danger btn--sm"
                          onClick={() => unpair(d.id)}
                          disabled={busyId === d.id}
                        >
                          {busyId === d.id ? "Unpairing…" : "Unpair"}
                        </button>
                        <button
                          className="btn btn--ghost btn--sm"
                          onClick={() => setConfirmId(null)}
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <>
                        <span className="row-meta">
                          {active
                            ? "Online"
                            : isZeroTime(d.last_seen)
                              ? "Not seen yet"
                              : `Seen ${timeAgo(d.last_seen)}`}
                        </span>
                        <StatusDot tone={active ? "ok" : "idle"} />
                        <button
                          className="icon-btn icon-btn--danger"
                          aria-label={`Unpair ${d.name || shortId(d.id)}`}
                          onClick={() => setConfirmId(d.id)}
                        >
                          <Icon name="trash" size={15} />
                        </button>
                      </>
                    )}
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
