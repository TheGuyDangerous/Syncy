<div align="center">

# Syncy

**A fast, secure, local‑first peer‑to‑peer folder synchronization engine.**

Keep your folders in sync across Windows, macOS, Linux and Android — with no cloud, no subscription and no one in the middle. Your data stays on your devices.

[![CI](https://github.com/TheGuyDangerous/Syncy/actions/workflows/ci.yml/badge.svg)](https://github.com/TheGuyDangerous/Syncy/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/TheGuyDangerous/Syncy?include_prereleases&sort=semver&label=release)](https://github.com/TheGuyDangerous/Syncy/releases)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/engine-Go-00ADD8.svg?logo=go&logoColor=white)](https://go.dev)
[![Desktop](https://img.shields.io/badge/desktop-Tauri-24C8DB.svg?logo=tauri&logoColor=white)](https://tauri.app)
[![Mobile](https://img.shields.io/badge/mobile-Flutter-02569B.svg?logo=flutter&logoColor=white)](https://flutter.dev)
[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux%20%7C%20Android-4c1.svg)](#platform-support)

</div>

> [!NOTE]
> **Status: early development (pre‑alpha).** Syncy is being built in the open, milestone by milestone. This README documents the target architecture and tracks what is actually implemented — see the [feature status](#features) and [roadmap](./ROADMAP.md). Features are only ticked once they are implemented **and tested**.
>
> **Implemented so far:** the engine core (SHA‑256 content identifiers, content‑defined chunking, SQLite metadata store), a folder scanner with rename/move/delete‑aware index diffing, a native recursive filesystem watcher with a live folder monitor, and per‑device **Ed25519 identity with mutual TLS** — all unit‑tested and green on Windows, macOS and Linux CI. The **desktop app** (Tauri) shell builds into native installers. **Next up:** the DeltaSync Protocol over QUIC, then wiring the engine into the desktop and mobile clients.

---

## Why Syncy?

Most "sync" tools ask you to trust a company's servers with your files, pay a monthly fee, or both. Syncy takes the opposite approach:

- **Local‑first & peer‑to‑peer.** Every device runs the same synchronization engine and talks directly to your other devices. There is no cloud, no hosted API and no recurring cost.
- **Private by design.** Transfers are end‑to‑end encrypted. Only devices you have explicitly paired can exchange data.
- **Efficient at scale.** Block‑level (delta) synchronization means a one‑byte change in a large file transfers a few kilobytes, not the whole file. Metadata is exchanged before data so millions of files can be reconciled quickly.
- **Resilient.** If a device is offline, pending operations are queued and applied automatically once it reconnects. Interrupted transfers resume where they left off.
- **One engine, many clients.** The core is a single Go engine embedded by a desktop app (Tauri) and a mobile app (Flutter), and designed to also power a CLI, a headless server, NAS and Docker deployments later — without a rewrite.

Syncy is inspired by the strengths of Syncthing, Resilio Sync and Dropbox, but built as a clean, modern, fully open‑source platform.

## Features

Legend: ✅ implemented & tested · 🚧 in progress · ⬜ planned

### Core synchronization
- ⬜ Folder synchronization (one‑way & two‑way)
- ⬜ Automatic device discovery (LAN‑first, mDNS)
- ⬜ Device pairing & secure authentication
- ⬜ Native filesystem watchers (no polling)
- ⬜ Real‑time and manual synchronization
- ⬜ Offline operation queue with automatic replay
- ⬜ Resumable, integrity‑verified transfers
- ⬜ Block‑level / chunk‑level delta sync
- ⬜ Conflict detection & resolution
- ⬜ Delete, rename and move detection
- ⬜ Multi‑device synchronization

### Advanced
- ⬜ Version history with rollback
- ⬜ Visual live sync graph (devices, speeds, latency, queue)
- ⬜ Adaptive chunk sizing & bandwidth optimization
- ⬜ Smart compression (skip already‑compressed formats)
- ⬜ Optional AI assistance (BYOK) — conflict explanations, log analysis, troubleshooting

### Apps
- ⬜ Desktop app (Windows / macOS / Linux) with system‑tray background service
- ⬜ Mobile app (Android) — see the [`flutter`](https://github.com/TheGuyDangerous/Syncy/tree/flutter) branch

## Architecture

Syncy is a **synchronization engine with multiple clients**. The engine is a self‑contained Go core with clearly separated responsibilities; the apps are thin clients that embed or talk to it.

```
                         ┌───────────────────────────────────────────┐
                         │              Syncy Engine (Go)            │
                         │                                           │
   Filesystem  ─────────▶│  Filesystem Watcher                       │
                         │        │                                  │
                         │        ▼                                  │
                         │   Sync Engine ──▶ Conflict Resolver       │
                         │        │      └──▶ Version Manager         │
                         │        ▼                                  │
                         │  DeltaSync Protocol (DSP)  ──▶ Encryption │
                         │        │                                  │
                         │        ▼                                  │
                         │  Transfer Engine ──▶ Device Discovery     │
                         │        │                                  │
                         │        ▼                                  │
                         │   Metadata DB (SQLite)                    │
                         └───────────────────────────────────────────┘
                                    ▲                    ▲
                                    │  local control API │
                        ┌───────────┴──────┐   ┌─────────┴──────────┐
                        │  Desktop (Tauri) │   │  Mobile (Flutter)  │
                        └──────────────────┘   └────────────────────┘

                        Future clients: CLI · Headless server · NAS · Docker · Web dashboard
```

Each box is a package with a single responsibility. See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full design and the [DeltaSync Protocol](./docs/protocol/dsp.md) specification.

## Tech stack

| Layer            | Technology                                             |
| ---------------- | ------------------------------------------------------ |
| Sync engine      | **Go**                                                 |
| Transport        | **QUIC** (primary), gRPC for local control where useful |
| Database         | **SQLite**                                             |
| Hashing          | **SHA‑256** + content‑defined chunking                 |
| Compression      | **zstd / gzip** (only when it helps)                   |
| Security         | **TLS 1.3**, per‑device Ed25519 identities, E2E encryption |
| Desktop UI       | **Tauri** (Rust + web front‑end)                       |
| Mobile UI        | **Flutter** (Android first)                            |

## Repository layout

```
Syncy/
├── engine/            # Go synchronization engine (the core)
├── desktop/           # Tauri desktop application (planned)
├── docs/              # Architecture & protocol documentation
├── .github/           # CI/CD workflows, issue & PR templates
└── ...                # Project docs (README, LICENSE, ROADMAP, …)
```

The Android app is maintained on the [`flutter`](https://github.com/TheGuyDangerous/Syncy/tree/flutter) branch of this repository.

## Platform support

| Platform | Engine | Desktop app | Mobile app |
| -------- | :----: | :---------: | :--------: |
| Windows  |   ⬜   |     ⬜      |     —      |
| macOS    |   ⬜   |     ⬜      |     —      |
| Linux    |   ⬜   |     ⬜      |     —      |
| Android  |   ⬜   |      —      |     ⬜     |

Windows↔Windows, Windows↔macOS, Windows↔Linux, macOS↔Linux and Android↔desktop are all target combinations.

## Getting started

> Build instructions will expand as the engine and apps come online. For now:

```bash
# Clone
git clone https://github.com/TheGuyDangerous/Syncy.git
cd Syncy
```

Early pre‑release builds of the engine daemon (`syncyd`) for Windows, macOS and Linux are published on the [Releases](https://github.com/TheGuyDangerous/Syncy/releases) page. Developer setup, prerequisites and per‑component build steps live in [CONTRIBUTING.md](./CONTRIBUTING.md); how releases are cut is documented in [docs/RELEASING.md](./docs/RELEASING.md).

## Roadmap

Syncy is built in vertical milestones — each one is implemented, tested, documented and merged before the next begins. See [ROADMAP.md](./ROADMAP.md) for the full plan and current progress.

## Contributing

Contributions are very welcome. Please read [CONTRIBUTING.md](./CONTRIBUTING.md) and our [Code of Conduct](./CODE_OF_CONDUCT.md) before opening a pull request. Found a security issue? See [SECURITY.md](./SECURITY.md).

## License

Syncy is licensed under the [Mozilla Public License 2.0](./LICENSE).
