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
