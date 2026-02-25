# GHSA-pv8c-p6jf-3fpp Security Context Analysis

Advisory: `GHSA-pv8c-p6jf-3fpp` (draft, severity: critical)
Vulnerable range: `0.1.1`

## Trust Boundaries

1. **Channel ingress** — webhook/channel authentication.
2. **Agent tool boundary** — LLM can invoke `exec`, `cron`, `web_fetch`, etc.
3. **Network egress** — `web_fetch` SSRF exposure.
4. **Persistence** — cron jobs, session/state file permissions.

## Attack Chains Addressed

| Chain | Path | Mitigation |
|-------|------|------------|
| A | Ingress → agent → `exec` → host RCE | `exec` blocked for remote channels by default (`AllowRemote: false`). Fail-closed: empty channel = blocked. |
| B | Prompted SSRF via `web_fetch` | Private/loopback/link-local/metadata IP blocked. Safe dial context (connect-time DNS check). Redirect-to-private blocked. IPv6 vectors: unique local, 6to4, Teredo. |
| C | RCE → persistent cron commands | `cron` command scheduling restricted to internal channels + `command_confirm` friction. |
| D | File permission leakage | Session files: 0o600. State dir: 0o700, state files: 0o600. Atomic write via temp+rename. |

## Residual Risk (Not In This PR)

- `install_skill` suspicious content is warning-only (follow-up: block by default).
- Hardware `confirm` flags are not auth boundaries.
- Channel auth coverage outside WeCom not fully audited.
- Prompt injection via fetched web content remains an open vector.

## CWE Mapping

- CWE-306: Missing auth on critical function
- CWE-78: OS command injection
- CWE-918: SSRF
- CWE-276: Incorrect file permissions

## References

- [Advisory](https://github.com/sipeed/picoclaw/security/advisories/GHSA-pv8c-p6jf-3fpp)
- [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
