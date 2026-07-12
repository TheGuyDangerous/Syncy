# Syncy Architecture

Syncy is designed as **a synchronization engine with multiple clients**, not as a monolithic desktop app. The engine is a self‑contained Go core; the desktop (Tauri) and mobile (Flutter) apps are thin clients that embed or talk to it over a local control API. This separation is what lets the same engine later power a CLI, a headless server, NAS boxes, Docker and a web dashboard without a rewrite.

## Design principles

1. **One responsibility per package.** No god packages, no unnecessary abstractions.
2. **Local‑first & offline‑first.** Every core feature works with no internet and no API keys.
3. **Privacy by default.** Files never leave your devices; transfers are end‑to‑end encrypted.
4. **Never trust the network.** All wire input is validated and authenticated before use.
5. **Deterministic, testable cores.** I/O (disk, network, clock) is injected so logic can be unit‑tested.
6. **Extensibility over features.** Optional capabilities (e.g. AI) are plugins the core does not depend on.

## Layered overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                            Clients                                     │
│   Desktop (Tauri)   Mobile (Flutter)   [future: CLI, server, web]      │
└───────────────▲──────────────────────────────────────▲────────────────┘
                │ local control API (gRPC / HTTP over loopback)          │
┌───────────────┴──────────────────────────────────────┴────────────────┐
│                          Syncy Engine (Go)                             │
│                                                                        │
│  ┌────────────┐   ┌───────────────┐   ┌──────────────┐                 │
│  │ fswatch    │──▶│ sync engine   │──▶│ conflict     │                 │
│  │ (watcher + │   │ (orchestrator)│   │ resolver     │                 │
│  │  scanner)  │   │               │──▶│ versioning   │                 │
│  └────────────┘   └──────┬────────┘   └──────────────┘                 │
│                          │                                             │
│                          ▼                                             │
│  ┌────────────┐   ┌───────────────┐   ┌──────────────┐                 │
│  │ chunker /  │   │ DSP protocol  │──▶│ security     │                 │
│  │ hashing    │◀──│ (framing,     │   │ (TLS, E2E,   │                 │
│  │ (SHA-256)  │   │  reconcile)   │   │  identity)   │                 │
│  └────────────┘   └──────┬────────┘   └──────────────┘                 │
│                          │                                             │
│                          ▼                                             │
│  ┌────────────┐   ┌───────────────┐   ┌──────────────┐                 │
│  │ metadata   │◀─▶│ transfer      │◀─▶│ transport    │                 │
│  │ DB (SQLite)│   │ engine        │   │ (QUIC)       │                 │
│  └────────────┘   └───────────────┘   └──────┬───────┘                 │
│                                              │                         │
│                        ┌─────────────────────┴──────┐                  │
│                        │ discovery (mDNS LAN-first,  │                  │
│                        │ global fallback)            │                  │
│                        └────────────────────────────┘                  │
└────────────────────────────────────────────────────────────────────────┘
```

## Packages (target layout)

All engine code lives under `engine/`. Public, client‑facing APIs live under `engine/pkg`; everything else is `engine/internal`.

| Package                  | Responsibility |
| ------------------------ | -------------- |
| `internal/hashing`       | SHA‑256 file & block hashing; strong content identifiers. |
| `internal/chunker`       | Content‑defined chunking (rolling hash) for block‑level delta sync. |
| `internal/metadata`      | SQLite‑backed store: folders, files, blocks, devices, versions, queue. |
| `internal/fswatch`       | Native filesystem watchers + initial scanner + index diffing. |
| `internal/scanner`       | Walks a folder, produces a `FileIndex`; detects renames/moves via block hashes. |
| `internal/protocol`      | DeltaSync Protocol (DSP): message types, framing, reconciliation. |
| `internal/transport`     | QUIC transport, connection lifecycle, streams, backpressure. |
| `internal/transfer`      | Transfer engine: block scheduling, resume, parallelism, retries, integrity. |
| `internal/syncengine`    | Orchestrates scan → diff → reconcile → transfer → apply. |
| `internal/conflict`      | Version‑vector based conflict detection & resolution strategies. |
| `internal/versioning`    | Version history, snapshots and rollback. |
| `internal/discovery`     | mDNS LAN discovery + optional global discovery fallback. |
| `internal/security`      | Device identity (Ed25519), TLS 1.3 config, pairing, E2E encryption. |
| `internal/queue`         | Durable offline operation queue with automatic replay. |
| `pkg/api`                | Local control API surface consumed by the desktop/mobile clients. |
| `cmd/syncyd`             | The headless daemon that wires everything together. |

## Data model (metadata DB)

The SQLite database is the source of truth for local state. Core tables (evolving):

- **`devices`** — known/paired devices: id (from public key), name, trust state, addresses.
- **`folders`** — shared folders: id, label, local path, sync direction, ignore rules.
- **`files`** — per‑folder file index: path, size, mtime, permissions, version vector, deleted flag.
- **`blocks`** — content‑defined blocks: file id, offset, size, SHA‑256 hash (enables dedup & delta).
- **`versions`** — historical file versions for rollback.
- **`queue`** — pending operations awaiting a peer/connection.

## Synchronization flow (happy path)

1. **Watch/scan.** `fswatch` detects a local change (or the initial scan builds the index).
2. **Index.** `scanner` + `chunker` + `hashing` compute the file's blocks and update `metadata`.
3. **Announce.** The engine advertises an updated index summary to paired peers over DSP.
4. **Reconcile.** Peers exchange compact metadata to compute exactly which blocks each side is missing — no data is sent for unchanged content.
5. **Transfer.** `transfer` requests missing blocks over QUIC, in parallel, resuming on interruption, verifying each block's hash.
6. **Apply.** Blocks are assembled into a temporary file, the whole‑file hash is verified, and the file is atomically swapped into place. The prior version is retained by `versioning`.
7. **Resolve.** If both sides changed the same file concurrently, `conflict` detects it via version vectors and applies the configured resolution (or surfaces it to the user).

## Security model (summary)

- Each device has a long‑lived **Ed25519 identity**; its device ID is derived from the public key.
- Peers authenticate each other with **TLS 1.3** using these identities; only **paired, trusted** device IDs may connect.
- Application data is **end‑to‑end encrypted**; the transport additionally protects metadata.
- The protocol defends against **replay** (nonces/sequence), **directory traversal** (path validation), **tampering** (per‑block + per‑file hashing) and **malformed packets** (strict decoding with bounds checks).

See [SECURITY.md](./SECURITY.md) and the [DSP specification](./docs/protocol/dsp.md) for details.

## What is intentionally *not* here

- No cloud service, hosted API or account system.
- No dependency on any AI provider — AI is an optional plugin layer (disabled by default) that the engine never calls into for core functionality.
