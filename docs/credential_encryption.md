# Credential Encryption

PicoClaw supports encrypting `api_key` values in `model_list` configuration entries.
Encrypted keys are stored as `enc://<base64>` strings and decrypted automatically at startup.

---

## Quick Start

**1. Set your passphrase**

```bash
export PICOCLAW_KEY_PASSPHRASE="your-passphrase"
```

**2. Encrypt an API key**

Run `picoclaw onboard` — it prompts for your passphrase and generates the SSH key,
then automatically re-encrypts any plaintext `api_key` entries in your config on
the next `SaveConfig` call. The resulting `enc://` value will look like:

```
enc://AAAA...base64...
```

**3. Paste the output into your config**

```json
{
  "model_list": [
    {
      "model_name": "gpt-4o",
      "model": "openai/gpt-4o",
      "api_key": "enc://AAAA...base64...",
      "api_base": "https://api.openai.com/v1"
    }
  ]
}
```

---

## Supported `api_key` Formats

| Format | Example | Behaviour |
|--------|---------|-----------|
| Plaintext | `sk-abc123` | Used as-is |
| File reference | `file://openai.key` | Content read from the same directory as the config file |
| Encrypted | `enc://<base64>` | Decrypted at startup using `PICOCLAW_KEY_PASSPHRASE` |
| Empty | `""` | Passed through unchanged (used with `auth_method: oauth`) |

---

## Cryptographic Design

### Key Derivation

Encryption uses **HKDF-SHA256** with an SSH private key as a second factor.

```
sshHash = SHA256(ssh_private_key_file_bytes)
ikm     = HMAC-SHA256(key=sshHash, message=passphrase)
aes_key = HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
```

### Encryption

```
AES-256-GCM(key=aes_key, nonce=random[12], plaintext=api_key)
```

### Wire Format

```
enc://<base64( salt[16] + nonce[12] + ciphertext )>
```

| Field | Size | Description |
|-------|------|-------------|
| `salt` | 16 bytes | Random per encryption; fed into HKDF |
| `nonce` | 12 bytes | Random per encryption; AES-GCM IV |
| `ciphertext` | variable | AES-256-GCM ciphertext + 16-byte authentication tag |

The GCM authentication tag is appended to the ciphertext automatically. Any tampering causes decryption to fail with an error rather than returning corrupt plaintext.

### Performance

| Operation | Time (ARM Cortex-A) |
|-----------|---------------------|
| Key derivation (HKDF) | < 1 ms |
| AES-256-GCM decrypt | < 1 ms |
| **Total startup overhead** | **< 2 ms per key** |

---

## Two-Factor Security with SSH Key

When a SSH private key is provided, breaking the encryption requires **both**:

1. The **passphrase** (`PICOCLAW_KEY_PASSPHRASE`)
2. The **SSH private key file**

This means a leaked config file alone is not sufficient to recover the API key, even if the passphrase is weak. The SSH key contributes 256 bits of entropy (Ed25519) regardless of passphrase strength.

### Threat Model

| Attacker Has | Can Decrypt? |
|---|---|
| Config file only | No — needs passphrase + SSH key |
| SSH key only | No — needs passphrase |
| Passphrase only | No — needs SSH key |
| Config file + SSH key + passphrase | Yes — full compromise |

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PICOCLAW_KEY_PASSPHRASE` | Yes (for `enc://`) | Passphrase used for key derivation |
| `PICOCLAW_SSH_KEY_PATH` | No | Path to SSH private key. If not set, auto-detects from `~/.ssh/picoclaw_ed25519.key` |

### SSH Key Auto-Detection

If `PICOCLAW_SSH_KEY_PATH` is not set, PicoClaw looks for the picoclaw-specific key:

```
~/.ssh/picoclaw_ed25519.key
```

This dedicated file avoids conflicts with the user's existing SSH keys.
Run `picoclaw onboard` to generate it automatically.

`os.UserHomeDir()` is used for cross-platform home directory resolution (reads `USERPROFILE` on Windows, `HOME` on Unix/macOS).

> **Note:** An SSH key file is required for credential encryption. If no key is found and `PICOCLAW_SSH_KEY_PATH` is not set, encryption/decryption will fail. Run `picoclaw onboard` to generate the key automatically.

---

## Migration

Because the only secret material is `PICOCLAW_KEY_PASSPHRASE` and the SSH private key file, migration is straightforward:

1. Copy the config file to the new machine.
2. Set `PICOCLAW_KEY_PASSPHRASE` to the same value.
3. Copy the SSH private key file to the same path (or set `PICOCLAW_SSH_KEY_PATH` to its new location).

No re-encryption is needed.

---

## Security Considerations

- **Both passphrase and SSH key are required.** The SSH key acts as a second factor — without it, encryption/decryption will fail. Run `picoclaw onboard` to generate the key if it doesn't exist.
- **The SSH key is read-only at runtime.** PicoClaw never writes to or modifies the SSH key file.
- **Plaintext keys remain supported.** Existing configs without `enc://` are unaffected.
- **The `enc://` format is versioned** via the HKDF `info` field (`picoclaw-credential-v1`), allowing future algorithm upgrades without breaking existing encrypted values.
