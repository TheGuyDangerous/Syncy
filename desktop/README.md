# Syncy Desktop

The Syncy desktop application, built with [Tauri](https://tauri.app) (Rust shell) and a React + TypeScript front-end. It runs the Syncy engine in the background and provides the tray and window UI.

## Develop

```bash
npm install
npm run tauri dev
```

## Build installers

```bash
npm run tauri build
```

Installers are written to `src-tauri/target/release/bundle/` — NSIS `.exe` and MSI on Windows, `.dmg` on macOS, `.deb`/`.AppImage` on Linux.

## Prerequisites

Rust (stable), Node 18+, and the [Tauri platform prerequisites](https://tauri.app/start/prerequisites/) for your OS (on Windows: the MSVC build tools and WebView2).
