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
- Release pipeline now builds **native installers** with Tauri for Windows
  (NSIS/MSI), macOS (`.dmg`) and Linux (`.deb`/`.AppImage`) and attaches them to
  the GitHub pre-release, replacing the earlier raw engine archives.
