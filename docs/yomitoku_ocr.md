# yomitoku OCR Integration

picoclaw は [yomitoku](https://github.com/kotaro-kinoshita/yomitoku) CLI と連携し、スキャン PDF や画像ベースの PDF からテキストを抽出します。

## Setup

`config.json` の `ocr` セクションで設定:

```json
{
  "ocr": {
    "command": "/path/to/yomitoku/.venv/bin/yomitoku",
    "args": [],
    "env": {},
    "timeout": 600,
    "reading_order": "auto"
  }
}
```

| フィールド | 説明 | デフォルト |
|---|---|---|
| `command` | yomitoku CLI のパス | (必須) |
| `args` | 追加の CLI 引数 | `[]` |
| `env` | 環境変数 (key-value) | `{}` |
| `timeout` | OCR タイムアウト (秒) | `600` |
| `reading_order` | デフォルトの読み順 | `"auto"` |

### reading_order の値

| 値 | 用途 |
|---|---|
| `auto` | yomitoku が自動判定 (デフォルト) |
| `right2left` | 縦書き文書 |
| `top2bottom` | 横書きの一般文書 |
| `left2right` | 横並びの帳票系 |

## Processing Flow

PDF ファイルが送られると以下の順序で処理されます:

1. **Phase 1 (キーワード待ち)**: PDF のみ送信された場合、5秒間フォローアップメッセージを待つ
2. **pdftotext fast path**: テキストレイヤーのある PDF は `pdftotext` で高速抽出 (figures 指定時はスキップ)
3. **yomitoku OCR**: テキスト抽出に失敗した場合、yomitoku で OCR 実行
4. **Phase 2 (バッファリング)**: OCR 中のメッセージをバッファし、完了後に LLM に渡す

## Chat Keywords

メッセージ中のキーワードで OCR オプションを制御できます:

### Figures (図版抽出)

`--figure --figure_letter` を追加:

- `figure`, `figures`, `with images`
- `図版`, `図付き`, `画像付き`, `図も`

### Reading Order (読み順)

- `縦書き`, `たてがき`, `vertical` → `--reading_order right2left`
- `横書き`, `よこがき`, `horizontal` → `--reading_order top2bottom`
- `right2left`, `top2bottom`, `left2right` → そのまま指定

### Cancel (中断)

OCR 実行中に以下のキーワードで中断:

- `cancel`, `abort`, `stop`
- `中止`, `キャンセル`, `やめ`

## Output Structure

OCR 結果は `.ocr_cache/<hash>/` に保存されます:

```
.ocr_cache/
  c5a9f00fe7567ac0/     # FNV-1a 64bit hash (= cache key)
    document.md          # OCR テキスト (yomitoku 出力をリネーム)
    figures/             # 抽出された図版 (--figure 時)
  b92a7e9171b9b9e5/
    document.md          # pdftotext 結果
```

- ハッシュはキャッシュキーと同一 → ファイルからキャッシュを逆引き可能
- figures 有無・reading order が異なれば別ハッシュ (別ディレクトリ)

## Cache Management

- **自動 prune**: 7日間アクセスのないエントリを自動削除
- **Mini App**: キャッシュ一覧 + 個別削除 / 全削除 UI (`/miniapp/api/cache`)
- **API**: `DELETE /api/media-cache/{hash}` (個別) / `DELETE /api/media-cache` (全削除)
