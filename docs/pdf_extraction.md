# PDF Text Extraction

picoclaw はチャットで送られた PDF を自動的にテキスト化し、LLM のコンテキストに注入します。
テキストレイヤーのある PDF は高速な pdftotext で処理し、スキャン PDF は [yomitoku](https://github.com/kotaro-kinoshita/yomitoku) OCR にフォールバックします。

## Prerequisites

### pdftotext (推奨)

テキストレイヤーのある PDF を高速に処理します。OCR 不要な PDF はこちらだけで完結します。

```bash
# Ubuntu/Debian
sudo apt install poppler-utils

# macOS
brew install poppler
```

`pdftotext` と `pdfinfo` コマンドが PATH に必要です。

### yomitoku (スキャン PDF 用)

pdftotext で十分なテキストが得られない場合のフォールバック OCR エンジンです。
日本語文書 (縦書き含む) に特化しています。

```bash
pip install yomitoku
```

> **Note**: PyTorch 2.5+ (CUDA 11.8 以上) が必要です。GPU 環境を推奨。
> CUDA 環境がない場合は [Dockerfile](https://github.com/kotaro-kinoshita/yomitoku) を参照してください。

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

#### reading_order の値

| 値 | 用途 |
|---|---|
| `auto` | yomitoku が自動判定 (デフォルト) |
| `right2left` | 縦書き文書 |
| `top2bottom` | 横書きの一般文書 |
| `left2right` | 横並びの帳票系 |

## Processing Flow

PDF ファイルが送られると以下の順序で処理されます:

```
PDF 受信
  │
  ├─ テキストあり → そのまま処理開始
  │
  └─ PDF のみ (テキストなし) → Phase 1: 5秒間キーワード待ち
                                 │
                                 ▼
                            pdftotext で抽出を試行
                                 │
                          ┌──────┴──────┐
                          │             │
                     成功 (十分な       失敗 (テキスト少ない
                      テキスト量)       / 密度が低い)
                          │             │
                          ▼             ▼
                     テキスト保存    yomitoku OCR 実行
                                    (Phase 2: メッセージバッファリング)
                                        │
                                        ▼
                                    OCR 結果保存
```

### pdftotext Fast Path

- `pdftotext -layout` で抽出、`pdfinfo` でページ数を取得
- 判定基準: テキスト 100 文字以上 かつ 非空白文字密度 30% 以上
- figures キーワードが指定された場合はスキップ (図版は pdftotext では抽出できないため)

### yomitoku OCR

- pdftotext が失敗した場合、または figures キーワードが指定された場合に実行
- 進捗はスピナーで表示: `⠋ Processing PDF (3/12)...`
- 出力の .md ファイルは `document.md` にリネームされて保存

### Phase 1: キーワード待ち

PDF のみ (テキストなし) で送信された場合、5秒間フォローアップメッセージを待ちます。
この間に OCR オプションのキーワードを送ることができます。

### Phase 2: バッファリング

OCR 実行中に受信したメッセージはバッファされ、OCR 完了後に LLM コンテキストに追加されます。
`cancel` / `中止` で OCR を中断できます。

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

抽出結果は `.ocr_cache/<hash>/` に保存されます:

```
.ocr_cache/
  c5a9f00fe7567ac0/     # FNV-1a 64bit hash (= cache key)
    document.md          # OCR テキスト
    figures/             # 抽出された図版 (--figure 時のみ)
  b92a7e9171b9b9e5/
    document.md          # pdftotext テキスト
```

- ハッシュはキャッシュキーと同一 → ファイルからキャッシュを逆引き可能
- figures 有無・reading order が異なれば別ハッシュ (別ディレクトリ)
- LLM には先頭 500 文字のプレビュー + ファイルパスが渡され、`read_file` で全文を参照

## Cache Management

- **自動 prune**: 7日間アクセスのないエントリを自動削除
- **Mini App**: キャッシュ一覧 + 個別削除 / 全削除 UI
- **API**: `DELETE /api/media-cache/{hash}` (個別) / `DELETE /api/media-cache` (全削除)

## Troubleshooting

| 症状 | 原因 | 対処 |
|---|---|---|
| PDF を送っても反応しない | `pdftotext` / `pdfinfo` が未インストール、かつ OCR 未設定 | `poppler-utils` をインストール、または `ocr` を設定 |
| テキスト抽出が文字化け | PDF にテキストレイヤーがない (スキャン PDF) | yomitoku を設定 |
| OCR が遅い / タイムアウト | ページ数が多い、GPU なし | `timeout` を延長、GPU 環境を推奨 |
| 縦書きが正しく読めない | reading order が auto で誤判定 | `縦書き` キーワードを追加、または config で `reading_order: "right2left"` |
| 前回の PDF の結果が表示される | 旧形式のキャッシュが残っている | Mini App の Clear All でキャッシュを一掃 |
