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
