# Changelog — Syncy Mobile

All notable changes to the Android companion app are recorded here. The app is
versioned independently of the desktop engine and lives on the `flutter` branch.

## [Unreleased]

### Added
- Pairing flow: connect to a Syncy desktop by its LAN address and access token,
  with the connection persisted across launches.
- Home screen showing live sync status and the paired desktop, with
  pull-to-refresh.
- Folders screen listing the folders the desktop is syncing; a phone-side folder
  picker marked as a preview (nothing uploads yet).
- History screen for recent activity.
- Settings screen with connection details and a disconnect action.
- Material 3 dark theme sharing the desktop's status colour vocabulary and a
  monospace treatment for device identifiers.
- Continuous integration: analyze, test, and per-architecture debug APK builds
  (`--split-per-abi`) uploaded as artifacts on every push and pull request.
