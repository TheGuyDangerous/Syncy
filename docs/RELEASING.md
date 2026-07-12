# Releasing

Releases are produced automatically by the [`Release`](../.github/workflows/release.yml) workflow when a version tag is pushed.

## Cut a release

1. Make sure `main` is green.
2. Tag it and push the tag:

   ```bash
   git tag v0.0.1
   git push origin v0.0.1
   ```

3. The workflow cross-builds the `syncyd` engine daemon for Windows, macOS and Linux (amd64 and arm64), writes `SHA256SUMS.txt`, and publishes a GitHub Release with the archives attached.

Tags matching `v0.*`, or containing a pre-release suffix such as `-alpha`, are published as **pre-releases**.

## Versioning

Syncy will follow [Semantic Versioning](https://semver.org) from 1.0 onward. Until then, `0.x` builds are pre-releases and may change at any time. As the desktop and mobile apps come online, their installers and APKs are added to the same release pipeline.
