# Security Policy

Security is a first‑class concern for Syncy: it moves your files directly between your devices, so getting this right matters.

## Supported versions

Syncy is in early development. Until the first tagged release, only the latest `main` is supported for security fixes.

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, use GitHub's private vulnerability reporting:

1. Go to the repository's **Security** tab.
2. Click **Report a vulnerability**.
3. Provide a description, reproduction steps and impact assessment.

We aim to acknowledge reports within a few days and will keep you updated on remediation. Please give us a reasonable window to fix an issue before any public disclosure. We're happy to credit reporters who wish to be acknowledged.

## Threat model & mitigations

Syncy is designed to resist a network attacker who can observe, modify, replay or inject traffic between devices, as well as unauthorized devices attempting to join a sync group.

| Threat                       | Mitigation |
| ---------------------------- | ---------- |
| Unauthorized device access   | Devices must be explicitly **paired**; connections are authenticated with per‑device Ed25519 identities over TLS 1.3. |
| Eavesdropping                | All transfers are encrypted end‑to‑end; the QUIC transport encrypts metadata in transit. |
| Tampering / corruption       | Every block and every assembled file is verified against its SHA‑256 hash; corrupted transfers are retried. |
| Replay attacks               | Protocol messages carry nonces / monotonic sequence numbers; stale or duplicated frames are rejected. |
| Directory traversal          | All incoming paths are validated and confined to the shared folder root before any filesystem operation. |
| Malformed / hostile packets  | The wire decoder enforces strict bounds and length checks; malformed frames are dropped without affecting the process. |

## Scope

The core engine, its protocol, and the official desktop/mobile clients are in scope. Optional third‑party AI integrations (BYOK) run against endpoints you configure and are outside the sync trust boundary; data is only sent to them when you explicitly initiate an AI‑assisted action.

## Known advisories

- **`InsecureSkipVerify` in the TLS config (CodeQL `go/disabled-certificate-check`) — intentional, not a vulnerability.** Syncy authenticates peers by pinning the Ed25519 **device ID** derived from a self‑signed certificate, not by CA‑chain validation. So the TLS config sets `InsecureSkipVerify` *and* supplies a mandatory `VerifyPeerCertificate` callback that performs the device‑ID pinning and trust check — the standard pattern for self‑signed, key‑pinned peers. A peer cannot present a trusted device's ID without possessing that device's private key (TLS requires it to complete the handshake), so the default check is *replaced*, not removed; the peer is still authenticated. These CodeQL alerts are dismissed as false positives.
- **`glib` < 0.20.0 — `VariantStrIter` unsoundness (moderate).** The desktop app's Linux WebKitGTK backend pulls in `glib` 0.18 transitively through Tauri (`tauri` → `gtk`/`webkit2gtk` → `glib "^0.18"`). The fix ships in `glib` 0.20, which requires the gtk‑rs 0.20 stack that Tauri/webkit2gtk‑rs have not yet adopted, so no compatible upgrade exists in the dependency graph at this time. This affects **Linux builds only** — Windows uses WebView2 and macOS uses WKWebView, neither of which compiles `glib`. The affected code path (`GVariant` string‑array iteration) is not exercised by Syncy directly, and the impact is a potential crash. We track this and will bump as soon as Tauri adopts gtk‑rs 0.20; the engine and its network‑facing code are unaffected.
