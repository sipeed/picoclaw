> 返回 [README](../../README.zh.md)

# 凭据加密

PicoClaw 支持对 `model_list` 配置条目中的 `api_key` 值进行加密。
加密后的密钥以 `enc://<base64>` 字符串形式存储，并在启动时自动解密。

---

## 快速开始

**1. 设置密码短语**

```bash
export PICOCLAW_KEY_PASSPHRASE="your-passphrase"
```

**2. 加密 API 密钥**

运行 `picoclaw onboard` — 它会提示你输入密码短语并生成 SSH 密钥，
然后在下一次 `SaveConfig` 调用时自动重新加密配置中所有明文 `api_key` 条目。生成的 `enc://` 值如下所示：

```
enc://AAAA...base64...
```

**3. 将输出粘贴到你的配置中**

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

## 支持的 `api_key` 格式

| 格式 | 示例 | 行为 |
|------|------|------|
| 明文 | `sk-abc123` | 直接使用 |
| 文件引用 | `file://openai.key` | 从配置文件所在目录读取内容 |
| 加密 | `enc://<base64>` | 启动时使用 `PICOCLAW_KEY_PASSPHRASE` 解密 |
| 空值 | `""` | 原样传递（用于 `auth_method: oauth`） |

---

## 加密设计

### 密钥派生

加密使用 **HKDF-SHA256**，并以 SSH 私钥作为第二因子。

```
sshHash = SHA256(ssh_private_key_file_bytes)
ikm     = HMAC-SHA256(key=sshHash, message=passphrase)
aes_key = HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
```

### 加密

```
AES-256-GCM(key=aes_key, nonce=random[12], plaintext=api_key)
```

### 传输格式

```
enc://<base64( salt[16] + nonce[12] + ciphertext )>
```

| 字段 | 大小 | 描述 |
|------|------|------|
| `salt` | 16 字节 | 每次加密随机生成；输入 HKDF |
| `nonce` | 12 字节 | 每次加密随机生成；AES-GCM IV |
| `ciphertext` | 可变 | AES-256-GCM 密文 + 16 字节认证标签 |

GCM 认证标签会自动附加到密文之后。任何篡改都会导致解密失败并报错，而不是返回损坏的明文。

### 性能

| 操作 | 耗时 (ARM Cortex-A) |
|------|---------------------|
| 密钥派生 (HKDF) | < 1 ms |
| AES-256-GCM 解密 | < 1 ms |
| **启动总开销** | **每个密钥 < 2 ms** |

---

## 使用 SSH 密钥的双因子安全

当提供 SSH 私钥时，破解加密需要**同时具备**：

1. **密码短语** (`PICOCLAW_KEY_PASSPHRASE`)
2. **SSH 私钥文件**

这意味着仅泄露配置文件不足以恢复 API 密钥，即使密码短语较弱也是如此。SSH 密钥贡献 256 位熵（Ed25519），与密码短语强度无关。

### 威胁模型

| 攻击者拥有 | 能否解密？ |
|------------|-----------|
| 仅配置文件 | 否 — 需要密码短语 + SSH 密钥 |
| 仅 SSH 密钥 | 否 — 需要密码短语 |
| 仅密码短语 | 否 — 需要 SSH 密钥 |
| 配置文件 + SSH 密钥 + 密码短语 | 是 — 完全泄露 |

---

## 环境变量

| 变量 | 是否必需 | 描述 |
|------|----------|------|
| `PICOCLAW_KEY_PASSPHRASE` | 是（用于 `enc://`） | 用于密钥派生的密码短语 |
| `PICOCLAW_SSH_KEY_PATH` | 否 | SSH 私钥路径。如未设置，自动从 `~/.ssh/picoclaw_ed25519.key` 检测 |

### SSH 密钥自动检测

如果未设置 `PICOCLAW_SSH_KEY_PATH`，PicoClaw 会查找专用密钥：

```
~/.ssh/picoclaw_ed25519.key
```

此专用文件避免与用户现有的 SSH 密钥冲突。
运行 `picoclaw onboard` 可自动生成该密钥。

`os.UserHomeDir()` 用于跨平台主目录解析（在 Windows 上读取 `USERPROFILE`，在 Unix/macOS 上读取 `HOME`）。

> **注意：** SSH 密钥文件是凭据加密的必要条件。如果未找到密钥且未设置 `PICOCLAW_SSH_KEY_PATH`，加密/解密将失败。运行 `picoclaw onboard` 可自动生成密钥。

---

## 迁移

由于唯一的密钥材料是 `PICOCLAW_KEY_PASSPHRASE` 和 SSH 私钥文件，迁移非常简单：

1. 将配置文件复制到新机器。
2. 将 `PICOCLAW_KEY_PASSPHRASE` 设置为相同的值。
3. 将 SSH 私钥文件复制到相同路径（或将 `PICOCLAW_SSH_KEY_PATH` 设置为新位置）。

无需重新加密。

---

## 安全注意事项

- **密码短语和 SSH 密钥都是必需的。** SSH 密钥作为第二因子 — 没有它，加密/解密将失败。如果密钥不存在，运行 `picoclaw onboard` 生成。
- **SSH 密钥在运行时为只读。** PicoClaw 不会写入或修改 SSH 密钥文件。
- **仍然支持明文密钥。** 不使用 `enc://` 的现有配置不受影响。
- **`enc://` 格式通过版本控制**，通过 HKDF `info` 字段（`picoclaw-credential-v1`）实现，允许未来升级算法而不破坏现有加密值。
