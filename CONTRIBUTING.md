# Contributing to Syncy

Thanks for your interest in improving Syncy! This document explains how the project is organized and how to get a development environment running.

## Ground rules

- Be respectful — see the [Code of Conduct](./CODE_OF_CONDUCT.md).
- Keep pull requests **small and focused**. One logical change per PR is much easier to review than a large mixed one.
- Every change should leave the repository in a **better, still‑building, still‑tested** state.
- Add or update **tests** and **documentation** alongside code.
- Never suppress a linter/analyzer warning without a written justification.

## Repository & branches

| Branch      | Purpose |
| ----------- | ------- |
| `main`      | Stable, always‑buildable trunk. Releases are tagged here. |
| `dev`       | Integration branch for the engine and desktop app. PRs target `dev`, then merge to `main`. |
| `flutter`   | The Flutter Android app. Mobile PRs target this branch. |

Typical flow: branch from `dev` → implement → open a PR into `dev`/`main` → review → merge.

## Prerequisites

You only need the toolchain for the part you're working on.

| Component       | Requires |
| --------------- | -------- |
| Engine (`engine/`)  | Go 1.24+ |
| Desktop (`desktop/`)| Rust (stable) + Node 18+ + the Tauri prerequisites for your OS |
| Mobile (`flutter` branch) | Flutter 3.24+ (Dart 3.5+) + Android SDK |

## Building & testing the engine

```bash
cd engine
go build ./...
go test ./...
go vet ./...
gofmt -l .        # should print nothing
```

We use `golangci-lint` in CI. To run it locally:

```bash
golangci-lint run ./...
```

## Commit style

Write clear, conventional commit messages in the imperative mood, e.g.:

```
engine/chunker: add rolling-hash content-defined chunking

Implements a Rabin-style chunker with configurable min/avg/max block
sizes so a small edit to a large file only re-transfers the affected
blocks. Includes table-driven tests for boundary stability.
```

Reference issues where relevant (`Fixes #123`).

## Pull request checklist

Before requesting review, confirm:

- [ ] `go build ./...` and `go test ./...` pass (for engine changes).
- [ ] `gofmt`/`golangci-lint` are clean.
- [ ] New behavior is covered by tests, including edge cases.
- [ ] Public behavior changes are reflected in the docs and, if user‑facing, the [CHANGELOG](./CHANGELOG.md).
- [ ] The PR description explains the *why*, not just the *what*.

## Reporting bugs & requesting features

Please use the issue templates. For security vulnerabilities, **do not open a public issue** — follow [SECURITY.md](./SECURITY.md).
