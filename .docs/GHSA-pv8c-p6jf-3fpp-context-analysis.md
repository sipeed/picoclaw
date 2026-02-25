# GHSA-pv8c-p6jf-3fpp Security Context Analysis

Date: 2026-02-25
Repository: `sipeed/picoclaw`
Advisory: `GHSA-pv8c-p6jf-3fpp` (draft)

## 1) Advisory Snapshot (What Is Known)

From `gh api repos/sipeed/picoclaw/security-advisories/GHSA-pv8c-p6jf-3fpp`:

- Summary: `Unauthenticated RCE and multiple vulnerabilities in PicoClaw`
- Severity: `critical`
- State: `draft`
- Vulnerable range: `0.1.1`
- Impact text references shell execution, filesystem tools, web fetch, skill install, channel auth, I2C/SPI, cron, subagent, OAuth, and session management.

Source:
- https://github.com/sipeed/picoclaw/security/advisories/GHSA-pv8c-p6jf-3fpp

Important caveat:
- The advisory is still draft and does not yet publish patched versions, CVSS vectors, or a complete vulnerability decomposition.

## 2) Threat Actors And Trust Boundaries

### Threat actors

1. Unauthenticated external attacker reaching webhook/channel ingress.
2. Authenticated but malicious user abusing tool-capable agent behavior.
3. Adversarial remote content source (web pages consumed via `web_fetch`).
4. Supply-chain adversary through registry content (`install_skill`).
5. Local host user/process with filesystem access (lower priority in this GHSA).

### Core trust boundaries

1. Channel/webhook ingress authentication.
2. Agent decision boundary (LLM can invoke privileged tools).
3. Tool boundary (`exec`, `cron.command`, `install_skill`, hardware writes).
4. Network egress boundary (`web_fetch` SSRF class).
5. Persistence boundary (cron jobs, session/state files).

## 3) Code-Backed Security Context

This section separates verified facts from inference.

### 3.1 Verified: ingress-to-tool execution chain exists

- Channel messages are published into bus and consumed by agent loop:
  - `pkg/channels/base.go:84`
  - `pkg/channels/base.go:98`
  - `pkg/bus/bus.go:24`
  - `pkg/agent/loop.go:157`
  - `pkg/agent/loop.go:165`
- `exec` tool is registered on agent instances:
  - `pkg/agent/instance.go:54`
- Unix command execution uses shell:
  - `pkg/tools/shell.go:182` (`sh -c`)

Security implication:
- Any ingress auth failure can become command-execution impact if tool policy is insufficient.

### 3.2 Verified: WeCom signature fail-open class has been hardened

- Signature check now rejects empty token/signature/timestamp/nonce:
  - `pkg/channels/wecom.go:487`
  - `pkg/channels/wecom.go:488`
- WeCom App channel constructor now requires token:
  - `pkg/channels/wecom_app.go:121`
  - `pkg/channels/wecom_app.go:123`

Security implication:
- Closes a concrete empty-secret auth bypass condition for WeCom paths.

### 3.3 Verified: SSRF controls added for `web_fetch`

- `web_fetch` is exposed to agents:
  - `pkg/agent/loop.go:112`
- Private/local host checks:
  - `pkg/tools/web.go:505`
  - `pkg/tools/web.go:631`
- Redirect target checks:
  - `pkg/tools/web.go:531`
  - `pkg/tools/web.go:535`

Security implication:
- Basic SSRF hardening exists.

Open verification gap:
- Need explicit confirmation that checks are enforced at connect-time (dial-time) to resist DNS rebinding/TOCTOU, not only pre-resolution checks.

### 3.4 Verified: cron can persist shell command execution

- Cron supports scheduling a shell command:
  - `pkg/tools/cron.go:54`
  - `pkg/tools/cron.go:71`
  - `pkg/tools/cron.go:181`
  - `pkg/tools/cron.go:206`
  - `pkg/tools/cron.go:284`
- Cron store persistence permission:
  - `pkg/cron/service.go:343` (`0600`)

Security implication:
- Command execution can be made persistent once attacker reaches tool invocation.

### 3.5 Verified: skill install is a supply-chain boundary

- Registry download/install path:
  - `pkg/tools/skills_install.go:118`
- Malware blocked, suspicious currently warning-only:
  - `pkg/tools/skills_install.go:134`
  - `pkg/tools/skills_install.go:163`

Security implication:
- Warning-only suspicious policy is weak in automated LLM tool-calling context.

### 3.6 Verified: hardware write "confirm" is not an authorization boundary

- I2C/SPI write-like operations rely on `confirm` parameter:
  - `pkg/tools/i2c_linux.go:218`
  - `pkg/tools/spi_linux.go:70`

Security implication:
- `confirm` is model-supplied input; it reduces accidental writes but does not protect against adversarial prompting or ingress compromise.

### 3.7 Verified: session persistence permissions hardened

- Session save temp file mode:
  - `pkg/session/manager.go:217` (`0600`)

## 4) Attack/Vulnerability Chains

Non-operational risk chains for analysis.

### Chain A: Unauthenticated ingress -> tool invocation -> host command execution

1. Attacker reaches unauthenticated/weakly authenticated channel ingress.
2. Message flows into agent loop.
3. LLM invokes `exec`.
4. Command runs in host context.

Impact:
- Remote code execution.

### Chain B: Prompted SSRF via `web_fetch`

1. Adversary induces fetch of attacker-selected URL.
2. Tool accesses internal resources through direct or redirect flow.
3. Returned content is exposed to model and/or user.

Impact:
- Internal service discovery, metadata leakage, token/secret exposure.

### Chain C: RCE-to-persistence via `cron.command`

1. Initial execution foothold obtained.
2. Scheduled commands created through cron tool.
3. Commands run later out-of-band.

Impact:
- Durable persistence and repeated post-exploitation.

### Chain D: Supply-chain through `install_skill`

1. Adversarial skill selected/installed from registry.
2. Artifact persists in workspace.
3. Future workflow/tool usage can be influenced.

Impact:
- Long-lived compromise path or latent privilege abuse.

### Chain E: Indirect prompt injection through fetched content

1. `web_fetch` imports untrusted text into model context.
2. Malicious content instructs model to call privileged tools.
3. Agent executes commands/actions absent robust policy gating.

Impact:
- Tool-abuse without direct channel compromise.

## 5) What / Why / How (Mitigation Interpretation)

### What is implemented now

1. WeCom signature verification fail-closed behavior.
2. WeCom App token requirement.
3. `web_fetch` private/redirect host restrictions.
4. Session file permission tightening (`0600`).

### Why this helps

- Reduces direct unauthenticated ingress abuse in WeCom.
- Reduces straightforward SSRF to loopback/private network.
- Reduces local confidentiality exposure of session artifacts.

### How this is still insufficient (current residual risk)

1. Channel auth coverage outside WeCom is not yet fully documented as audited.
2. `exec` remains high-impact and currently denylist-driven.
3. `confirm` flags are not true auth controls.
4. Suspicious skill installs are warning-only.
5. SSRF rebinding/TOCTOU protection needs explicit verification.
6. Prompt injection via fetched content is an explicit abuse path.

## 6) Priority Recommendations

### P0 (must address for network-exposed deployments)

1. Complete channel-by-channel ingress auth audit and fail-closed startup for missing critical secrets.
2. Enforce fail-closed policy on `exec` for remote channels by default.
3. Validate SSRF protection against DNS rebinding/TOCTOU (connect-time checks) and metadata IP ranges.
4. Do not treat `confirm` as security control; add policy/approval gate for dangerous tools.

### P1 (high-value next controls)

1. Shift from denylist-heavy exec filtering to constrained allowlist profiles.
2. Block suspicious skill installs by default with explicit operator override.
3. Add auditable operator-visible controls for cron command creation/removal.

### P2 (defense-in-depth)

1. Channel-level abuse throttling/rate limits.
2. Expanded security telemetry for tool invocation chains and anomaly patterns.
3. Prompt-injection resilience policy for web-ingested content.

## 7) CWE Mapping

- Missing auth on critical function: CWE-306
- OS command execution/injection class: CWE-78
- SSRF: CWE-918
- Blacklist weakness: CWE-184
- Incorrect file permissions: CWE-276

## 8) References

1. GitHub advisory:
   - https://github.com/sipeed/picoclaw/security/advisories/GHSA-pv8c-p6jf-3fpp
2. GitHub webhook signature validation guidance:
   - https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
3. OWASP SSRF prevention cheat sheet:
   - https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
4. CWE entries:
   - https://cwe.mitre.org/data/definitions/306.html
   - https://cwe.mitre.org/data/definitions/78.html
   - https://cwe.mitre.org/data/definitions/918.html
   - https://cwe.mitre.org/data/definitions/184.html
   - https://cwe.mitre.org/data/definitions/276.html
5. Go APIs:
   - https://pkg.go.dev/os#WriteFile
   - https://pkg.go.dev/crypto/subtle#ConstantTimeCompare
