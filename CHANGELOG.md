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
- Desktop bundles the `syncyd` engine as a **Tauri sidecar**: a build script
  compiles the Go daemon for the host target triple, the app spawns it on
  startup pointing at a per-user data directory, and a `daemon_info` command
  exposes the local control API base URL and token to the front-end.
- Desktop UI foundation: a premium, dark‑first (light‑capable) interface — a
  typed control‑API client with graceful offline handling, an app shell with a
  sidebar (Dashboard, Devices, Folders, Transfers, History, Versions, Conflicts,
  Logs, Settings), a status‑colour design system, and working Dashboard,
  Folders (add/remove), Devices, Conflicts and Settings (theme, AI‑provider
  BYOK, about) screens wired to the daemon.
- Desktop **system tray** with a menu (Open, Quit) and left‑click to open the
  window, plus **close‑to‑tray**: closing the window hides it and keeps the
  engine running in the background, so only Quit fully exits.
- Desktop **launch at login**: a "Start Syncy at login" toggle in Settings →
  General registers/unregisters the app to start in the background at sign‑in.
- Desktop **Versions** screen: pick a folder and a file path to browse the
  recovered earlier copies of that file (timestamp, size, when it changed),
  wired to the engine's version history. Completes the desktop app (M6).
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

### Fixed
- The control API now answers CORS preflight requests and sends
  `Access-Control-Allow-Origin`, so the desktop app's web view can actually read
  the engine's responses. Without it, a healthy engine still showed as
  "offline" because the browser blocked every cross-origin request to the
  loopback API. The API stays loopback-only and token-authenticated.
- The desktop app no longer gets stuck on "Can't reach the sync engine" when the
  engine's peer port is already in use. The daemon now brings up its local
  control API **first** and treats the QUIC peer listener as best-effort: a busy
  P2P port pauses syncing until restart instead of taking the whole engine — and
  the UI — down with it.
- The app cleans up a stale engine left by a previous run (tracked by PID) before
  starting a new one, and now runs **single-instance**, so relaunching focuses
  the existing window instead of spawning a second, port-conflicting engine.
- The dashboard retries the connection every 1.5s while the engine is still
  starting, so the first launch settles on its own instead of stalling on
  "Engine offline".

### Security
- Hardened against **path traversal** (CodeQL `go/path-injection`): a new `fsafe`
  helper rejects non-local paths (absolute or `..`-escaping), and the sync
  session and version store now validate every incoming file path before any
  filesystem operation, so a malicious peer or request can't read or write
  outside a shared folder. Makes the directory-traversal protection real, not
  just claimed.
- Documented that the TLS `InsecureSkipVerify` CodeQL alerts are false positives:
  peers are authenticated by pinning the Ed25519 device ID via a mandatory
  `VerifyPeerCertificate`, not by CA-chain validation.
- Documented the `glib` < 0.20 `VariantStrIter` advisory (moderate) as a known
  issue: it reaches us only transitively through Tauri's Linux WebKitGTK stack,
  has no compatible upstream fix yet (Tauri pins gtk-rs 0.18), affects Linux
  desktop builds only, and does not touch the engine. See SECURITY.md.
- Resolved the `golang.org/x/crypto` advisories by upgrading to 0.52.
- Release pipeline now builds **native installers** with Tauri for Windows
  (NSIS/MSI), macOS (`.dmg`) and Linux (`.deb`/`.AppImage`) and attaches them to
  the GitHub pre-release, replacing the earlier raw engine archives.
- Release builds now set up Go and produce a universal macOS engine sidecar
  (`aarch64` + `x86_64` via `lipo`), so tagged installer builds no longer fail
  on the sidecar step.
- **Unified releases:** every pre-release now carries both the desktop
  installers and the Android APKs. Each side is versioned independently and
  built only when it changes — the other side's artifacts are carried over from
  the previous release, so a desktop release never rebuilds the mobile app and
  vice versa. See `RELEASING.md`.
- CI gained a desktop job that builds the engine sidecar, compiles the frontend,
  and type-checks the Tauri shell on every push and pull request.
- `ai` package: an optional, bring-your-own-key assistant that can explain a
  sync conflict and summarize engine logs. It speaks the OpenAI, Anthropic and
  Gemini API shapes, covering OpenAI, OpenRouter, Ollama, LM Studio, a custom
  OpenAI-compatible endpoint (all one shape), Anthropic and Gemini. It is
  disabled by default and fully isolated — the sync engine never depends on it.
  Tested against mocked providers with no network or real keys.
- Control API gained AI routes: read/save the provider config (the API key is
  never returned and is kept across saves), test a connection, explain a
  conflict, and analyze logs. Config is persisted to `ai.json` in the data dir.
- Desktop Settings → Integrations now configures the AI assistant against the
  engine: an enable toggle, a provider picker (OpenAI, Anthropic, Gemini,
  OpenRouter, Ollama, LM Studio, custom), model and optional base URL, a
  password key field that shows "saved" without echoing the key, and a Test
  Connection button. The Conflicts screen gained a per-file **Explain** action
  that asks the assistant why a file conflicts and how to resolve it.
- Custom window frame: the desktop app draws its own titlebar (brand, drag
  region, minimize / maximize / close, resizable edges) instead of the native OS
  chrome, so the window is themed with the app in both light and dark. Close
  still hides to the tray.
- New app icon — a linked-peers mark on a dark ground — shared across the desktop
  installers and the Android app.
