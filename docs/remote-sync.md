# Remote sync — friends over the internet

> Status: **experimental (alpha)**. Shipped engine‑side; the desktop and mobile
> UIs for it land in a later phase.

Syncy devices have always found each other on the local network via mDNS. Remote
sync extends this so two devices on *different* networks can pair and sync —
**serverless**: there is no rendezvous, relay or account infrastructure. Devices
talk directly or not at all.

## How pairing works

1. **Invite code.** A device produces a compact code (`GET /invite`) containing
   its device ID, name and the endpoints it believes are reachable: its LAN
   addresses plus, when internet discovery is enabled and the router cooperates,
   an external `host:port` mapped via UPnP‑IGD or NAT‑PMP.
2. **Friend request.** The other device pastes the code (`POST /friends`). The
   engine dials the listed endpoints over QUIC, proves its own identity, pins
   the dialed connection to the device ID from the code, and sends a
   `FriendRequest` with its ID, name and endpoints. The receiver stores it as a
   pending request — nothing more.
3. **Acceptance.** The receiving user accepts
   (`POST /friend-requests/{id}/accept`), which marks the requester trusted,
   saves its endpoints, and — when reachable — sends back
   `FriendResponse{accepted}`. The requester marks the acceptor trusted on
   receipt. If the response cannot be delivered right away, the requester's
   periodic re‑send of the friend request completes the exchange later: a peer
   that already trusts the sender answers a `FriendRequest` with an immediate
   acceptance.
4. **Sync.** Both sides now trust each other and know each other's endpoints.
   The daemon periodically dials trusted devices' saved endpoints and runs the
   normal DeltaSync reconciliation, exactly as it does for mDNS‑discovered LAN
   peers.

## Security model

- **Identity is proven at the transport layer, always.** Connections are mutual
  TLS 1.3; each side's device ID is derived from the Ed25519 key in its
  certificate. This is unchanged.
- **Trust is enforced at the application layer.** Any device with a valid
  identity may *connect*, but an untrusted peer may open exactly one stream and
  send exactly one thing: a `FriendRequest` (or a `FriendResponse` answering a
  request we made). Everything else gets an `Error` frame. Folder indexes and
  blocks are never served to untrusted peers.
- A `FriendRequest` must carry the same device ID the TLS handshake proved,
  otherwise it is rejected.
- A `FriendResponse` only creates trust if we previously sent that exact device
  a friend request (`pending_outgoing`); unsolicited acceptances are ignored.
- Outgoing dials pin the expected device ID, so a hijacked address or DNS entry
  cannot impersonate a friend.

## Discovery settings

`GET/PUT /settings/discovery` controls two independent switches, persisted in
the metadata store:

| Setting    | Default | Effect |
| ---------- | ------- | ------ |
| `local`    | on      | mDNS announce/browse and automatic LAN sync (applied at startup). |
| `internet` | off     | Attempt a UPnP/NAT‑PMP port mapping (refreshed periodically) and periodically dial trusted devices' saved endpoints. |

## Reach — honest limitations

Remote sync is serverless, so it reaches peers that are **addressable**:

- at least one side has a public IP, a manual port‑forward, or a router that
  grants a UPnP‑IGD/NAT‑PMP mapping with a public external address;
- mappings whose "external" address is itself private (double NAT, CGNAT) are
  detected and discarded rather than advertised.

If **both** sides sit behind CGNAT or mapping‑refusing NATs, there is currently
no path between them. NAT hole‑punching is deliberately **not** faked; a future
optional relay/rendezvous service is the honest fix and is tracked as follow‑up
work. Also out of scope for now: multi‑seeder swarming — transfers use the
existing single seeder → receiver block pull.
