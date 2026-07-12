# Roadmap

Syncy is built in vertical milestones. Each milestone is implemented, tested, documented and merged before the next one begins — no half‑finished systems left lying around.

Legend: ✅ done · 🚧 in progress · ⬜ planned

## M0 — Repository foundation ✅
Project scaffolding: README, architecture, license, contribution & security docs, CI skeleton, issue/PR templates. Cross‑OS CI is green.

## M1 — Engine core ✅
Go module and `syncyd` skeleton; SHA‑256 content identifiers (`hashing`); deterministic content‑defined chunking for block‑level delta sync (`chunker`); shared domain types (`core`); SQLite metadata store with migrations and device/folder persistence (`metadata`, pure‑Go, no cgo). All unit‑tested and green on Windows, macOS and Linux.

## M2 — Filesystem watcher & scanner ✅
Native recursive watcher (fsnotify, no polling); folder scanner producing a content‑addressed index; deterministic index diffing with rename/move/delete detection; a folder monitor that streams live change sets. All tested and green on Windows, macOS and Linux.

## M3 — DeltaSync Protocol & transport ✅
Ed25519 device identity with mutual TLS; authenticated QUIC transport; the DeltaSync Protocol framing and messages; and a reconciliation session that syncs real files between two peers — pulling missing files, reusing local blocks by hash (delta + dedup), verifying block and whole‑file integrity, and atomically replacing files. Tested with two‑node loopback sync and malformed‑frame handling.

## M4 — Sync engine, conflicts & versioning ✅
Bidirectional convergence over a connection; version history with rollback; version‑vector and last‑synced‑baseline conflict detection that writes conflict copies instead of clobbering; a durable offline queue; and an `Engine` facade with `Sync` that ties it together. Persisted baselines make conflict detection real across sync rounds.

## M5 — Discovery & daemon ✅
mDNS LAN‑first discovery; a token‑authenticated local HTTP control API; and the `syncyd` daemon that ties it together — identity, metadata, QUIC listener (trusted peers only), discovery‑driven sync, and the control API, with graceful shutdown.

## M6 — Desktop app (Tauri) ✅
Premium, dark‑first UI wired to the daemon: the engine ships as a bundled sidecar; a system‑tray background service with close‑to‑tray; a sidebar app shell; Dashboard, Devices, Folders (add/remove), Conflicts, Versions and Settings (theme, AI‑provider BYOK, launch‑at‑startup) screens; Transfers/History/Logs as placeholders pending live feeds.

## M7 — Mobile app (Flutter, Android) 🚧 (next)
Companion app on the `flutter` branch: pairing, folder selection (media/downloads/documents/custom), manual + automatic + background sync, progress, history, notifications.

## M8 — CI/CD & packaging ⬜
Cross‑OS build & test matrix; installers for Windows/macOS/Linux; Flutter APK/AAB; release automation on tags.

## M9 — Optional AI plugin (BYOK) ⬜
Pluggable, disabled‑by‑default AI layer: conflict explanations, log analysis, troubleshooting. Providers include OpenAI, Anthropic, Gemini, OpenRouter, Ollama, LM Studio and custom OpenAI‑compatible endpoints. The core never depends on it.

## Beyond
CLI client · headless server mode · NAS & Docker images · web dashboard · iOS · selective sync · plugin architecture.
