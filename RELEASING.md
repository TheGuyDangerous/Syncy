# Releasing Syncy

Syncy ships as a single stream of GitHub pre-releases. Every release carries the
**full set** of artifacts — desktop installers for Windows, macOS and Linux, and
the Android APKs — so a user finds everything in one place.

## How versioning works

Three numbers move independently:

- **Desktop version** — `desktop/src-tauri/tauri.conf.json` (`version`). Baked
  into the installer filenames (e.g. `Syncy_0.1.0_x64-setup.exe`).
- **Mobile version** — `pubspec.yaml` (`version`) on the `flutter` branch. Baked
  into the APK filenames (e.g. `syncy-mobile-1.0.0-arm64-v8a.apk`).
- **Release tag** — the number you push (e.g. `v0.3`). It increments every
  release and is what the release is titled by.

When only the mobile app changes, its version bumps, the desktop version stays
the same, and the release tag moves forward. The generated release notes say
which side changed and which was carried over unchanged.

## Cutting a release

The branch you tag decides which side is built. `main` and `flutter` are
independent histories, so a tag points at one of them and only that side runs.

**Desktop release** (a desktop or engine change shipped):

```bash
git checkout main && git pull
git tag v0.3 && git push origin v0.3
```

Runs `release-desktop.yml`: builds the installers, then carries the latest APKs
from the previous release into the new one.

**Mobile release** (a mobile change shipped):

```bash
git checkout flutter && git pull
git tag v0.4 && git push origin v0.4
```

Runs `release-mobile.yml`: builds the APKs, then carries the latest desktop
installers from the previous release into the new one.

## Why it stays fast

Each release rebuilds only the side that changed. The other side's binaries are
downloaded from the most recent release that has them — that release is the
cache — and re-attached. A mobile-only release never spins up the three desktop
runners; a desktop-only release never builds the APKs.
