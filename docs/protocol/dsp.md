# DeltaSync Protocol (DSP) — design specification

> Status: **draft**. This document describes the target design of DSP. It will
> evolve alongside the implementation in `engine/internal/protocol`.

DSP is Syncy's application‑layer synchronization protocol. Its job is to let two
paired devices reconcile the state of a shared folder and exchange only the data
that is actually missing — efficiently, securely and resumably — even across
folders containing millions of files.

## Goals

- **Metadata before data.** Devices exchange compact indexes to determine
  exactly which blocks are needed *before* any file content moves.
- **Delta, not full copies.** Unchanged blocks are never retransmitted.
- **Resumable.** A transfer interrupted at any point continues from where it
  stopped, not from the beginning.
- **Verifiable.** Every block and every reconstructed file is integrity‑checked.
- **Secure.** All frames travel over an authenticated, encrypted channel; the
  protocol resists replay, tampering, traversal and malformed input.
- **Scalable.** Index exchange is incremental so large trees don't require
  re‑sending the whole index on every change.

## Transport

DSP runs over **QUIC** (TLS 1.3). QUIC gives us:

- Encrypted, authenticated streams out of the box.
- Stream multiplexing without head‑of‑line blocking, so metadata and many block
  transfers proceed concurrently over a single connection.
- Connection migration, which helps on mobile networks.

Each logical exchange uses its own QUIC stream:

- **Control stream** (bidirectional): handshake, index announcements, requests.
- **Data streams** (one per in‑flight block batch): raw block payloads.

## Device identity & handshake

- Every device owns a long‑lived **Ed25519** key pair. Its **device ID** is
  derived from the public key.
- The QUIC/TLS handshake authenticates both peers by these identities: the
  connection fails unless the remote cryptographically proves a device ID.
- **Trust is enforced at the application layer.** An identity‑verified but
  *untrusted* peer may open exactly one stream carrying a `FriendRequest` (or a
  `FriendResponse` answering a request we sent); any other message is answered
  with an `Error` frame and no folder metadata or data is ever served to it.
  Only devices marked trusted reach the reconciliation flow below.
- Pairing establishes mutual trust either out of band (exchanging device IDs)
  or in band via invite codes and the friend‑request exchange described in
  [remote sync](../remote-sync.md).

## Message framing

Every DSP message on the control stream is a length‑prefixed frame:

```
+--------+---------+-----------------+----------------------+
| type   | flags   | length (uint32) | payload (length B)   |
| 1 byte | 1 byte  | 4 bytes, BE     | ...                  |
+--------+---------+-----------------+----------------------+
```

The decoder enforces a maximum frame length and validates every field before
allocating or acting on it. Unknown message types are ignored (forward
compatibility); malformed frames terminate the stream.

## Message types (initial set)

| Type            | Direction | Purpose |
| --------------- | --------- | ------- |
| `Hello`         | both      | Announce protocol version, device ID, capabilities. |
| `FolderSummary` | both      | Advertise a shared folder and a digest of its index. |
| `IndexUpdate`   | both      | Incremental list of changed files/blocks since a marker. |
| `BlockRequest`  | A → B     | Request a set of blocks by (file, offset, hash). |
| `BlockData`     | B → A     | Deliver a requested block's bytes (verified by hash). |
| `Ack`           | both      | Acknowledge applied updates / delivered blocks. |
| `Ping`/`Pong`   | both      | Liveness and latency measurement. |
| `Error`         | both      | Structured, non‑fatal error notification. |
| `FriendRequest` | A → B     | Ask an identity‑verified peer to establish mutual trust; carries the sender's ID, name and endpoints. |
| `FriendResponse`| B → A     | Answer a friend request; `accepted` plus the responder's name and endpoints. |

## Reconciliation flow

1. **Hello.** Peers exchange versions and capabilities.
2. **Summaries.** Each peer sends a `FolderSummary` per shared folder, including
   a digest that lets the other side detect whether its view is up to date.
3. **Index sync.** If digests differ, peers exchange `IndexUpdate`s carrying only
   the entries changed since the last acknowledged marker.
4. **Need computation.** Each peer locally computes which blocks it is missing by
   comparing block hash sets — content‑addressed, so identical blocks anywhere
   (dedup, moved files) are never re‑fetched.
5. **Requests.** Missing blocks are requested with `BlockRequest`, prioritized
   and pipelined; large files are fetched block‑by‑block in parallel.
6. **Delivery & verify.** `BlockData` payloads are verified against their
   expected SHA‑256 before being written to a temporary assembly file.
7. **Finalize.** When a file's blocks are complete, its whole‑file hash is
   verified and it is atomically renamed into place; the previous version is
   retained by the versioning subsystem.
8. **Ack.** Applied updates are acknowledged so the marker advances and future
   index syncs stay incremental.

## Resume & reliability

- Partially received files persist their completed‑block bitmap, so a
  reconnection resumes only the outstanding blocks.
- Any block failing verification is re‑requested; repeated failures back off and
  surface an error rather than corrupting the target.

## Security properties

- **Replay protection:** frames within a session are ordered; stale/duplicate
  control frames are rejected.
- **Path safety:** every incoming path is normalized and confined to the shared
  folder root; `..` and absolute paths are rejected.
- **Integrity:** per‑block and per‑file SHA‑256 verification.
- **Confidentiality & authenticity:** provided by TLS 1.3 over QUIC between
  authenticated device identities, with end‑to‑end encryption of payloads.

## Open questions (tracked as the implementation matures)

- Exact index digest scheme (Merkle vs. running hash) for very large trees.
- Adaptive block size selection thresholds.
- Compression negotiation per file type.
