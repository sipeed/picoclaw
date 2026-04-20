> [README](../project/README.ja.md) に戻る

# PicoClaw で Antigravity プロバイダーを使用する

このガイドでは、PicoClaw で **Antigravity**（Google Cloud Code Assist）プロバイダーをセットアップして使用する方法を説明します。

## 前提条件

1.  Google アカウント。
2.  Google Cloud Code Assist が有効であること（通常「Gemini for Google Cloud」のオンボーディングから利用可能）。

## 1. 認証

Antigravity で認証するには、以下のコマンドを実行します：

```bash
picoclaw auth login --provider antigravity
```

### 手動認証（ヘッドレス/VPS）
サーバー（Coolify/Docker）上で実行しており、`localhost` にアクセスできない場合は、以下の手順に従ってください：
1.  上記のコマンドを実行します。
2.  表示された URL をコピーし、ローカルブラウザで開きます。
3.  ログインを完了します。
4.  ブラウザが `localhost:51121` URL にリダイレクトされます（ページは読み込めません）。
5.  **ブラウザのアドレスバーからその最終 URL をコピーします**。
6.  **PicoClaw が待機しているターミナルにそれを貼り付けます**。

PicoClaw が自動的に認証コードを抽出し、プロセスを完了します。

## 2. モデルの管理

### 利用可能なモデルの一覧
プロジェクトがアクセスできるモデルとそのクォータを確認するには：

```bash
picoclaw auth models
```

### モデルの切り替え
`~/.picoclaw/config.json` でデフォルトモデルを変更するか、CLI でオーバーライドできます：

```bash
# 単一コマンドでオーバーライド
picoclaw agent -m "Hello" --model claude-opus-4-6-thinking
```

## 3. 実際の使用方法（Coolify/Docker）

Coolify または Docker でデプロイしている場合、以下の手順でテストしてください：

1.  **環境変数**：
    *   `PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-flash`
2.  **認証の永続化**：
    ローカルでログイン済みの場合、認証情報をサーバーにコピーできます：
    ```bash
    scp ~/.picoclaw/auth.json user@your-server:~/.picoclaw/
    ```
    *または*、ターミナルアクセスがある場合、サーバー上で `auth login` コマンドを一度実行してください。

## 4. トラブルシューティング

*   **空のレスポンス**：モデルが空の応答を返す場合、プロジェクトで制限されている可能性があります。`gemini-3-flash` または `claude-opus-4-6-thinking` を試してください。
*   **429 レート制限**：Antigravity には厳格なクォータがあります。制限に達した場合、PicoClaw はエラーメッセージに「リセット時間」を表示します。
*   **404 Not Found**：`picoclaw auth models` リストのモデル ID を使用していることを確認してください。フルパスではなく、短い ID（例：`gemini-3-flash`）を使用してください。

## 5. 動作確認済みモデルのまとめ

テストに基づき、以下のモデルが最も信頼性が高いです：
*   `gemini-3-flash`（高速、高可用性）
*   `gemini-2.5-flash-lite`（軽量）
*   `claude-opus-4-6-thinking`（高性能、推論機能を含む）
