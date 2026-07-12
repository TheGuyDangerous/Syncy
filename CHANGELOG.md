# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it
reaches its first release.

## [Unreleased]

### Added
- Initial project foundation: README, architecture overview, roadmap, license
  (MPL‑2.0), contribution guide, security policy and code of conduct.
- Continuous integration skeleton and issue / pull‑request templates.
- DeltaSync Protocol (DSP) design notes.
- Go engine module bootstrap: the `syncyd` daemon entry point and a
  `buildinfo` package (with tests). CI now builds and tests the engine on
  Windows, macOS and Linux, running the race detector on Linux.
- `hashing` package: SHA-256 content identifiers (`Hash`) with helpers to hash
  bytes, strings, readers and files, plus hex parsing and text marshalling for
  storage. Verified against known SHA-256 test vectors.
- `chunker` package: deterministic content-defined chunking using a rolling
  gear hash with configurable Min/Avg/Max sizes. Streams with bounded memory
  and is shift-resistant, so a small edit only re-chunks nearby data — the
  basis for block-level delta sync. Tested for coverage, size bounds,
  determinism, streaming/byte parity and shift resistance.
- `core` package: shared domain types (`Device`, `Folder`, `SyncDirection`)
  used across the engine.
- `metadata` package: SQLite-backed store (pure-Go `modernc.org/sqlite`, no
  cgo) with a versioned migration runner and CRUD for devices and folders.
  WAL journaling, foreign keys and a busy timeout are enabled. Tested for
  migration idempotency, round-trips, upsert semantics and not-found paths.
- `hashing.Hasher`: an incremental `io.Writer` hasher, so a file can be
  whole-file hashed in the same pass it is chunked.
- `scanner` package: walks a folder into a content-addressed `Index` (per file:
  size, mtime, mode, whole-file hash and content-defined blocks). Paths are
  relative and slash-separated for cross-platform consistency; supports a skip
  predicate for ignore rules. Tested for correctness, determinism, empty files,
  skipping and error paths.
- `scanner.Diff`: compares two indexes into a deterministic change set
  (added / modified / deleted) and detects renames and moves by matching
  identical non-empty content, so a moved file transfers no data.
- `fswatch` package: a native, recursive filesystem watcher (fsnotify) that
  picks up new subdirectories and emits debounced batches of changed paths.
- `monitor` package: ties the watcher, scanner and diff together into a live
  stream of change sets for a folder, with a baseline index on startup. Tested
  end-to-end for create, delete and rename detection.
- `identity` package: per-device Ed25519 identity with a device ID derived from
  the public key, PKCS#8 persistence, a self-signed TLS certificate, and
  mutual-TLS configs that authenticate peers by pinning their device ID.
- Desktop application scaffold (`desktop/`): Tauri 2 shell with a React +
  TypeScript front-end and a dark, minimal window.
- `transport` package: authenticated QUIC transport (quic-go) built on the
  device identity's mutual-TLS configs. Connections expose the peer's device ID
  and multiplexed streams. Tested with a loopback handshake, peer-ID
  verification, bidirectional data exchange and peer rejection.
- `protocol` package: the DeltaSync Protocol (DSP) wire format — length-prefixed
  framing with a strict maximum frame size, the message types (Hello,
  FolderSummary, IndexUpdate, BlockRequest, BlockData, Ack, Ping/Pong, Error),
  and raw block-data encoding. Tested for round-trips and rejection of
  oversized, truncated and malformed frames.
- `session` package: DeltaSync reconciliation over the QUIC transport. A device
  serves its folder index and blocks and pulls missing files from a peer,
  reusing blocks it already has locally (delta + dedup by hash), verifying every
  block and the whole-file hash, then atomically replacing the file. Two-node
  loopback integration tests sync real files end-to-end, including delta reuse
  of shared blocks and skipping up-to-date files.
- `versioning` package: keeps recoverable, timestamped copies of files before
  they are overwritten or deleted, with listing, restore and pruning to a
  configurable maximum number of versions per file.
- `conflict` package: per-file version vectors that classify two file histories
  as equal, one ahead of the other, or concurrent (a real conflict), plus
  merge and conflict-copy naming.
- Durable offline operation queue in the metadata store (schema v2): pending
  operations for a device are persisted and can be listed, retried and
  completed, so work survives restarts and replays when a peer reconnects.
- The sync session can now archive a file into the version store before it is
  overwritten (`session.WithVersioning`), so an incoming change never silently
  discards the previous content.
- `syncengine` package: an `Engine` facade that manages folders and devices
  (persisted in the metadata store), and `Converge`, which serves the local
  folder while pulling the peer's so two devices bidirectionally reach the union
  of their newest files. Verified with a two-node convergence integration test.
- Conflict handling in the sync session (`WithBaseline` + `WithConflictNaming`):
  when both sides changed a file since the last sync, the incoming version is
  written as a `.sync-conflict-…` copy instead of overwriting local changes; a
  file only changed on one side fast-forwards, and a locally-ahead file is kept.
- Persisted last-synced baseline (metadata schema v3) wired into `Engine.Sync`,
  which scans a folder, converges with a peer using the baseline for conflict
  detection and version history, then records the new baseline. A two-round
  integration test verifies baselines persist and concurrent edits produce a
  conflict copy — completing the sync engine (M4).
- `api` package: a loopback HTTP + JSON control API with bearer-token auth,
  exposing status, folder management (list / add / remove), devices, conflicts
  and per-file version history — the surface the desktop and mobile clients
  drive. Includes `Engine.Conflicts` and `Engine.FolderVersions`.
- `discovery` package: LAN‑first device discovery over mDNS / DNS‑SD — a device
  advertises its ID and QUIC port and browses for peers, so devices find each
  other with no configuration.
- `session.Serve` now dispatches by the requested folder (`FolderSource`), so a
  single connection can serve every folder a device shares.
- `daemon` package and a wired `syncyd`: the engine runs as a real background
  service — it loads/creates the device identity and metadata store, listens for
  QUIC connections (accepting only trusted, paired devices), announces and
  discovers peers over mDNS, syncs shared folders with discovered trusted peers,
  and serves the token-authenticated control API. Graceful shutdown on
  SIGINT/SIGTERM; data directory, listen and API addresses are configurable.
  Completes the daemon and M5.

### Security
- Documented the `glib` < 0.20 `VariantStrIter` advisory (moderate) as a known
  issue: it reaches us only transitively through Tauri's Linux WebKitGTK stack,
  has no compatible upstream fix yet (Tauri pins gtk-rs 0.18), affects Linux
  desktop builds only, and does not touch the engine. See SECURITY.md.
- Resolved the `golang.org/x/crypto` advisories by upgrading to 0.52.
- Release pipeline now builds **native installers** with Tauri for Windows
  (NSIS/MSI), macOS (`.dmg`) and Linux (`.deb`/`.AppImage`) and attaches them to
  the GitHub pre-release, replacing the earlier raw engine archives.
