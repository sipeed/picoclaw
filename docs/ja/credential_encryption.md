> [README](../../README.ja.md) に戻る

# クレデンシャル暗号化

PicoClaw は `model_list` 設定エントリの `api_key` 値の暗号化をサポートしています。
暗号化されたキーは `enc://<base64>` 文字列として保存され、起動時に自動的に復号されます。

---

## クイックスタート

**1. パスフレーズを設定する**

```bash
export PICOCLAW_KEY_PASSPHRASE="your-passphrase"
```

**2. API キーを暗号化する**

`picoclaw onboard` を実行します — パスフレーズの入力を求められ、SSH キーが生成されます。
その後、次の `SaveConfig` 呼び出し時に、設定内のすべての平文 `api_key` エントリが自動的に再暗号化されます。生成される `enc://` 値は以下のようになります：

```
enc://AAAA...base64...
```

**3. 出力を設定に貼り付ける**

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

## サポートされる `api_key` 形式

| 形式 | 例 | 動作 |
|------|---|------|
| 平文 | `sk-abc123` | そのまま使用 |
| ファイル参照 | `file://openai.key` | 設定ファイルと同じディレクトリから内容を読み取り |
| 暗号化 | `enc://<base64>` | 起動時に `PICOCLAW_KEY_PASSPHRASE` を使用して復号 |
| 空 | `""` | そのまま渡される（`auth_method: oauth` で使用） |

---

## 暗号設計

### 鍵導出

暗号化には **HKDF-SHA256** を使用し、SSH 秘密鍵を第二要素とします。

```
sshHash = SHA256(ssh_private_key_file_bytes)
ikm     = HMAC-SHA256(key=sshHash, message=passphrase)
aes_key = HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
```

### 暗号化

```
AES-256-GCM(key=aes_key, nonce=random[12], plaintext=api_key)
```

### ワイヤーフォーマット

```
enc://<base64( salt[16] + nonce[12] + ciphertext )>
```

| フィールド | サイズ | 説明 |
|-----------|--------|------|
| `salt` | 16 バイト | 暗号化ごとにランダム生成；HKDF に入力 |
| `nonce` | 12 バイト | 暗号化ごとにランダム生成；AES-GCM IV |
| `ciphertext` | 可変 | AES-256-GCM 暗号文 + 16 バイト認証タグ |

GCM 認証タグは暗号文に自動的に付加されます。改ざんがあった場合、破損した平文を返すのではなく、エラーで復号が失敗します。

### パフォーマンス

| 操作 | 所要時間 (ARM Cortex-A) |
|------|------------------------|
| 鍵導出 (HKDF) | < 1 ms |
| AES-256-GCM 復号 | < 1 ms |
| **起動時の総オーバーヘッド** | **キーあたり < 2 ms** |

---

## SSH キーによる二要素セキュリティ

SSH 秘密鍵が提供されている場合、暗号を破るには**両方**が必要です：

1. **パスフレーズ** (`PICOCLAW_KEY_PASSPHRASE`)
2. **SSH 秘密鍵ファイル**

これは、設定ファイルが漏洩しただけでは、パスフレーズが弱い場合でも API キーを復元できないことを意味します。SSH キーはパスフレーズの強度に関係なく、256 ビットのエントロピー（Ed25519）を提供します。

### 脅威モデル

| 攻撃者が持っているもの | 復号可能か？ |
|----------------------|-------------|
| 設定ファイルのみ | いいえ — パスフレーズ + SSH キーが必要 |
| SSH キーのみ | いいえ — パスフレーズが必要 |
| パスフレーズのみ | いいえ — SSH キーが必要 |
| 設定ファイル + SSH キー + パスフレーズ | はい — 完全な侵害 |

---

## 環境変数

| 変数 | 必須 | 説明 |
|------|------|------|
| `PICOCLAW_KEY_PASSPHRASE` | はい（`enc://` 使用時） | 鍵導出に使用するパスフレーズ |
| `PICOCLAW_SSH_KEY_PATH` | いいえ | SSH 秘密鍵のパス。未設定の場合、`~/.ssh/picoclaw_ed25519.key` から自動検出 |

### SSH キーの自動検出

`PICOCLAW_SSH_KEY_PATH` が設定されていない場合、PicoClaw は専用キーを探します：

```
~/.ssh/picoclaw_ed25519.key
```

この専用ファイルにより、ユーザーの既存の SSH キーとの競合を回避します。
`picoclaw onboard` を実行すると自動的に生成されます。

`os.UserHomeDir()` はクロスプラットフォームのホームディレクトリ解決に使用されます（Windows では `USERPROFILE`、Unix/macOS では `HOME` を読み取ります）。

> **注意：** SSH キーファイルはクレデンシャル暗号化に必須です。キーが見つからず `PICOCLAW_SSH_KEY_PATH` も設定されていない場合、暗号化/復号は失敗します。`picoclaw onboard` を実行してキーを自動生成してください。

---

## 移行

唯一の秘密情報は `PICOCLAW_KEY_PASSPHRASE` と SSH 秘密鍵ファイルであるため、移行は簡単です：

1. 設定ファイルを新しいマシンにコピーします。
2. `PICOCLAW_KEY_PASSPHRASE` を同じ値に設定します。
3. SSH 秘密鍵ファイルを同じパスにコピーします（または `PICOCLAW_SSH_KEY_PATH` を新しい場所に設定します）。

再暗号化は不要です。

---

## セキュリティに関する考慮事項

- **パスフレーズと SSH キーの両方が必須です。** SSH キーは第二要素として機能します — これがなければ暗号化/復号は失敗します。キーが存在しない場合は `picoclaw onboard` を実行して生成してください。
- **SSH キーは実行時に読み取り専用です。** PicoClaw は SSH キーファイルへの書き込みや変更を行いません。
- **平文キーは引き続きサポートされます。** `enc://` を使用しない既存の設定は影響を受けません。
- **`enc://` 形式はバージョン管理されています。** HKDF `info` フィールド（`picoclaw-credential-v1`）により、既存の暗号化値を壊すことなく将来のアルゴリズムアップグレードが可能です。
