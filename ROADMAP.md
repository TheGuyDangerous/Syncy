# Roadmap

Syncy is built in vertical milestones. Each milestone is implemented, tested, documented and merged before the next one begins — no half‑finished systems left lying around.

Legend: ✅ done · 🚧 in progress · ⬜ planned

## M0 — Repository foundation ✅
Project scaffolding: README, architecture, license, contribution & security docs, CI skeleton, issue/PR templates. Cross‑OS CI is green.

## M1 — Engine core ✅
Go module and `syncyd` skeleton; SHA‑256 content identifiers (`hashing`); deterministic content‑defined chunking for block‑level delta sync (`chunker`); shared domain types (`core`); SQLite metadata store with migrations and device/folder persistence (`metadata`, pure‑Go, no cgo). All unit‑tested and green on Windows, macOS and Linux.

## M2 — Filesystem watcher & scanner ✅
Native recursive watcher (fsnotify, no polling); folder scanner producing a content‑addressed index; deterministic index diffing with rename/move/delete detection; a folder monitor that streams live change sets. All tested and green on Windows, macOS and Linux.

## M3 — DeltaSync Protocol & transport 🚧 (next)
Device identity (Ed25519); TLS 1.3 over QUIC; DSP message framing; metadata reconciliation; block request/response; resume; integrity verification; replay protection. Tests for malformed/corrupted packets.

## M4 — Sync engine, conflicts & versioning ⬜
Orchestration of scan → reconcile → transfer → apply; version‑vector conflict detection & resolution; version history with rollback; durable offline queue.

## M5 — Discovery & daemon ⬜
mDNS LAN‑first discovery with global fallback; the `syncyd` daemon; local control API (gRPC/HTTP) for clients.

## M6 — Desktop app (Tauri) ⬜
Premium, dark‑first UI: system‑tray background service with status colors, popover, dashboard, devices, folders, transfers, history, versions, conflicts, logs, settings (incl. Integrations). Close‑to‑tray; launch on startup.

## M7 — Mobile app (Flutter, Android) ⬜
Companion app on the `flutter` branch: pairing, folder selection (media/downloads/documents/custom), manual + automatic + background sync, progress, history, notifications.

## M8 — CI/CD & packaging ⬜
Cross‑OS build & test matrix; installers for Windows/macOS/Linux; Flutter APK/AAB; release automation on tags.

## M9 — Optional AI plugin (BYOK) ⬜
Pluggable, disabled‑by‑default AI layer: conflict explanations, log analysis, troubleshooting. Providers include OpenAI, Anthropic, Gemini, OpenRouter, Ollama, LM Studio and custom OpenAI‑compatible endpoints. The core never depends on it.

## Beyond
CLI client · headless server mode · NAS & Docker images · web dashboard · iOS · selective sync · plugin architecture.
