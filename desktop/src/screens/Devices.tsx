import { useEffect, useRef, useState, type FormEvent } from "react";
import { api, errorMessage, type Device, type FriendRequest } from "../lib/api";
import { useApi } from "../lib/useApi";
import { isRecent, isZeroTime, shortId, timeAgo } from "../lib/format";
import { Screen } from "../components/Screen";
import { Icon } from "../components/Icon";
import { StatusPill } from "../components/StatusDot";
import { EmptyState } from "../components/EmptyState";
import { Notice } from "../components/Notice";

interface Snapshot {
  invite: string;
  devices: Device[];
  requests: FriendRequest[];
}

async function loadFriends(): Promise<Snapshot> {
  const [invite, devices, requests] = await Promise.all([
    api.invite(),
    api.devices(),
    api.friendRequests(),
  ]);
  return { invite: invite.code, devices, requests };
}

export default function Devices() {
  const { data, error, loading, reload } = useApi(loadFriends);
  const [showAdd, setShowAdd] = useState(false);
  const [code, setCode] = useState("");
  const [addBusy, setAddBusy] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);
  const [addOk, setAddOk] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef<number | undefined>(undefined);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [reqBusy, setReqBusy] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    const t = window.setInterval(reload, 10000);
    return () => window.clearInterval(t);
  }, [reload]);

  function openAdd() {
    setShowAdd(true);
    setAddError(null);
    setAddOk(null);
  }

  function closeAdd() {
    setShowAdd(false);
    setAddError(null);
    setAddOk(null);
    setCode("");
  }

  async function copyCode() {
    if (!data) return;
    try {
      await navigator.clipboard.writeText(data.invite);
    } catch {
      const ta = document.createElement("textarea");
      ta.value = data.invite;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      ta.remove();
    }
    setCopied(true);
    window.clearTimeout(copyTimer.current);
    copyTimer.current = window.setTimeout(() => setCopied(false), 1500);
  }

  function submitAdd(e: FormEvent) {
    e.preventDefault();
    const trimmed = code.trim();
    if (!trimmed || addBusy) return;
    setAddBusy(true);
    setAddError(null);
    setAddOk(null);
    const knownFriends = new Set(
      (data?.devices ?? []).filter((d) => d.trusted).map((d) => d.id),
    );
    api
      .addFriend(trimmed)
      .then((res) => {
        setAddBusy(false);
        setCode("");
        const who = res.device.name || shortId(res.device.id);
        if (res.device.trusted) {
          setAddOk(
            knownFriends.has(res.device.id)
              ? `You're already friends with ${who}.`
              : `They'd already sent you a request — you're now friends with ${who}.`,
          );
        } else if (res.delivered) {
          setAddOk("Request sent — they'll need to accept on their device.");
        } else {
          setAddOk(
            "Request saved — their device isn't reachable right now. Syncy keeps retrying in the background.",
          );
        }
        reload();
      })
      .catch((err: unknown) => {
        setAddBusy(false);
        setAddError(errorMessage(err));
      });
  }

  function accept(id: string) {
    setReqBusy(id);
    setActionError(null);
    api
      .acceptFriendRequest(id)
      .then(() => {
        setReqBusy(null);
        reload();
      })
      .catch((err: unknown) => {
        setReqBusy(null);
        setActionError(errorMessage(err));
      });
  }

  function ignore(id: string) {
    setReqBusy(id);
    setActionError(null);
    api
      .rejectFriendRequest(id)
      .then(() => {
        setReqBusy(null);
        reload();
      })
      .catch((err: unknown) => {
        setReqBusy(null);
        setActionError(errorMessage(err));
      });
  }

  function unfriend(deviceId: string) {
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
        <button className="btn btn--primary" onClick={() => (showAdd ? closeAdd() : openAdd())}>
          <Icon name="plus" size={14} />
          Add a friend
        </button>
      }
    >
      {error ? <Notice message={error} onRetry={reload} /> : null}
      {actionError ? <Notice message={actionError} /> : null}

      {data ? (
        <section className="card">
          <div className="setting-row">
            <div>
              <p className="setting-name">Your invite code</p>
              <p className="setting-hint">
                Share this code with someone to connect. You'll both approve it once, then you're
                friends.
              </p>
              <p className="mono device-id" title={data.invite}>
                {data.invite}
              </p>
            </div>
            <button className="btn btn--sm" onClick={copyCode}>
              <Icon name="copy" size={13} />
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
        </section>
      ) : null}

      {showAdd ? (
        <form className="card form-card" onSubmit={submitAdd}>
          <label className="field">
            <span className="field-label">Their invite code</span>
            <input
              className="input input--mono"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="SYNCY1-…"
            />
          </label>
          <p className="setting-hint">
            Paste the code from their Devices screen. They'll get a request to approve.
          </p>
          {addError ? (
            <p className="form-error" role="alert">
              {addError}
            </p>
          ) : null}
          {addOk ? (
            <p className="form-ok" role="status">
              {addOk}
            </p>
          ) : null}
          <div className="form-actions">
            <button type="submit" className="btn btn--primary" disabled={addBusy || !code.trim()}>
              {addBusy ? "Sending…" : "Send request"}
            </button>
            <button type="button" className="btn btn--ghost" onClick={closeAdd}>
              {addOk ? "Done" : "Cancel"}
            </button>
          </div>
        </form>
      ) : null}

      {data && data.requests.length > 0 ? (
        <section className="card">
          <header className="card-head">
            <h2 className="card-title">Incoming requests</h2>
            <StatusPill
              tone="warn"
              label={
                data.requests.length === 1 ? "1 pending" : `${data.requests.length} pending`
              }
            />
          </header>
          <ul className="rows">
            {data.requests.map((r) => (
              <li key={r.from_id} className="row">
                <span className="row-icon">
                  <Icon name="devices" />
                </span>
                <div className="row-body">
                  <p className="row-title">{r.name || shortId(r.from_id)} wants to connect</p>
                  <p className="row-sub mono" title={r.from_id}>
                    {shortId(r.from_id)} · {timeAgo(r.created_at)}
                  </p>
                </div>
                <div className="row-end">
                  <button
                    className="btn btn--primary btn--sm"
                    onClick={() => accept(r.from_id)}
                    disabled={reqBusy === r.from_id}
                  >
                    {reqBusy === r.from_id ? "Working…" : "Accept"}
                  </button>
                  <button
                    className="btn btn--ghost btn--sm"
                    onClick={() => ignore(r.from_id)}
                    disabled={reqBusy === r.from_id}
                  >
                    Ignore
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {loading && !data ? <div className="loading">Loading…</div> : null}

      {data ? (
        <section className="card">
          <header className="card-head">
            <h2 className="card-title">Friends</h2>
            <span className="card-count mono">{data.devices.length}</span>
          </header>
          {data.devices.length === 0 ? (
            <EmptyState
              icon="devices"
              title="No friends yet"
              hint="Share your code or add someone's."
              action={
                !showAdd ? (
                  <button className="btn" onClick={openAdd}>
                    <Icon name="plus" size={14} />
                    Add a friend
                  </button>
                ) : undefined
              }
            />
          ) : (
            <ul className="rows">
              {data.devices.map((d) => {
                const pending = !d.trusted && d.pending_outgoing;
                const active = d.trusted && isRecent(d.last_seen);
                return (
                  <li key={d.id} className="row">
                    <span className="row-icon">
                      <Icon name="devices" />
                    </span>
                    <div className="row-body">
                      <p className="row-title">
                        {d.name || shortId(d.id)}
                        {pending ? <span className="pill">Invited</span> : null}
                      </p>
                      <p className="row-sub mono" title={d.id}>
                        {shortId(d.id)}
                      </p>
                    </div>
                    <div className="row-end">
                      {confirmId === d.id ? (
                        <>
                          <span className="row-meta">
                            {pending
                              ? "Cancel this request?"
                              : "Unfriend? Syncing with them stops."}
                          </span>
                          <button
                            className="btn btn--danger btn--sm"
                            onClick={() => unfriend(d.id)}
                            disabled={busyId === d.id}
                          >
                            {busyId === d.id ? "Removing…" : pending ? "Remove" : "Unfriend"}
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
                          {pending ? (
                            <StatusPill tone="warn" label="Waiting for them to accept" />
                          ) : active ? (
                            <StatusPill tone="ok" label="Online" />
                          ) : (
                            <StatusPill
                              tone="idle"
                              label={
                                isZeroTime(d.last_seen)
                                  ? "Not seen yet"
                                  : `Seen ${timeAgo(d.last_seen)}`
                              }
                            />
                          )}
                          <button
                            className="icon-btn icon-btn--danger"
                            aria-label={`Unfriend ${d.name || shortId(d.id)}`}
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
          )}
        </section>
      ) : null}
    </Screen>
  );
}
