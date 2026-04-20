# 🐛 トラブルシューティング

> [README](../project/README.ja.md) に戻る

## "model ... not found in model_list" または OpenRouter "free is not a valid model ID"

**症状：** 以下のいずれかのエラーが表示されます：

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter が 400 を返す：`"free is not a valid model ID"`

**原因：** `model_list` エントリの `model` フィールドは API に送信される値です。OpenRouter では省略形ではなく、**完全な**モデル ID を使用する必要があります。

- **誤り：** `"model": "free"` → OpenRouter は `free` を受け取り、拒否します。
- **正しい：** `"model": "openrouter/free"` → OpenRouter は `openrouter/free` を受け取ります（自動無料枠ルーティング）。

**修正方法：** `~/.picoclaw/config.json`（またはお使いの設定パス）で：

1. **agents.defaults.model_name** は `model_list` 内の `model_name` と一致する必要があります（例：`"openrouter-free"`）。
2. そのエントリの **model** は有効な OpenRouter モデル ID である必要があります。例：
   - `"openrouter/free"` – 自動無料枠
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

設定例：

```json
{
  "agents": {
    "defaults": {
      "model_name": "openrouter-free"
    }
  },
  "model_list": [
    {
      "model_name": "openrouter-free",
      "model": "openrouter/free",
      "api_key": "sk-or-v1-YOUR_OPENROUTER_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

キーは [OpenRouter Keys](https://openrouter.ai/keys) で取得できます。
