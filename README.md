# Syncy Mobile

The Android companion app for [Syncy](https://github.com/TheGuyDangerous/Syncy) — a local‑first, peer‑to‑peer folder synchronization engine.

This app lives on the `flutter` branch of the Syncy repository and is developed independently of the desktop/engine code on `main`.

## What it is

Syncy Mobile is a **companion** to a Syncy desktop device on your network. It connects to a desktop's local control API (over your LAN, authenticated with a token) to let you:

- pair with a desktop device (its address + access token),
- see live sync status and connected devices,
- browse the folders that device is syncing,
- review conflicts and a file's version history,
- choose which of the phone's folders to share (Photos, Downloads, Documents, custom).

## Scope (honest status)

Syncy's synchronization engine is written in Go and runs as a background service on the desktop. Running that full engine in‑process on a phone is a larger effort, so for this version the mobile app is a **controller / monitor** that talks to a desktop peer — not yet a standalone sync node. Phone‑side folder selection is captured in the UI; on‑device transfer over the DeltaSync protocol is planned. The README and changelog call out clearly what is wired to a real device versus previewed.

## Design

Same visual language as the desktop app: dark‑first, rounded, soft shadows, the shared sync‑status colour vocabulary (synced / syncing / pending / error / offline), and a monospace treatment for machine data like device IDs.

## Develop

```bash
flutter pub get
flutter run
```

Requires Flutter 3.24+ and the Android SDK. Build per-architecture debug APKs with `flutter build apk --debug --split-per-abi`.
