<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Go で書かれた超効率 AI アシスタント</h1>

  <h3>$10 ハードウェア · <10MB RAM · <1秒起動 · 行くぜ、シャコ！</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | **日本語** | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [English](README.md)

</div>

---

> **PicoClaw** は [Sipeed](https://sipeed.com) が立ち上げた独立したオープンソースプロジェクトです。完全に **Go 言語**で一から書かれており、OpenClaw、NanoBot、その他のプロジェクトのフォークではありません。

🦐 PicoClaw は [NanoBot](https://github.com/HKUDS/nanobot) にインスパイアされた超軽量パーソナル AI アシスタントです。Go でゼロからリファクタリングされ、AI エージェント自身がアーキテクチャの移行とコード最適化を推進するセルフブートストラッピングプロセスで構築されました。

⚡️ $10 のハードウェアで 10MB 未満の RAM で動作：OpenClaw より 99% 少ないメモリ、Mac mini より 98% 安い！

<table align="center">
  <tr align="center">
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/picoclaw_mem.gif" width="360" height="240">
      </p>
    </td>
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/licheervnano.png" width="400" height="240">
      </p>
    </td>
  </tr>
</table>

> [!CAUTION]
> **🚨 セキュリティ＆公式チャンネル**
>
> * **暗号通貨なし:** PicoClaw には公式トークン/コインは**一切ありません**。`pump.fun` やその他の取引プラットフォームでの主張はすべて**詐欺**です。
>
> * **公式ドメイン:** **唯一**の公式サイトは **[picoclaw.io](https://picoclaw.io)**、企業サイトは **[sipeed.com](https://sipeed.com)** です。
> * **注意:** 多くの `.ai/.org/.com/.net/...` ドメインは第三者によって登録されています。
> * **注意:** PicoClaw は初期開発段階にあり、未解決のネットワークセキュリティ問題がある可能性があります。v1.0 リリース前に本番環境へのデプロイは避けてください。
> * **注記:** PicoClaw は最近多くの PR をマージしており、最新バージョンではメモリフットプリントが大きくなる場合があります（10〜20MB）。機能セットが安定次第、リソース最適化を優先する予定です。

## 📢 ニュース

2026-03-17 🚀 **v0.2.3 リリース！** システムトレイ UI（Windows & Linux）、サブエージェントステータス追跡（`spawn_status`）、実験的ゲートウェイホットリロード、cron セキュリティゲート、セキュリティ修正 2 件。PicoClaw **25K ⭐** 達成！

2026-03-09 🎉 **v0.2.1 — 史上最大のアップデート！** MCP プロトコル対応、4 つの新チャネル（Matrix/IRC/WeCom/Discord Proxy）、3 つの新プロバイダー（Kimi/Minimax/Avian）、ビジョンパイプライン、JSONL メモリストア、モデルルーティング。

2026-02-28 📦 **v0.2.0** リリース — Docker Compose 対応と Web UI ランチャー。

2026-02-26 🎉 PicoClaw がわずか 17 日で **20K スター** 達成！チャネル自動オーケストレーションとケイパビリティインターフェースが実装されました。

<details>
<summary>過去のニュース...</summary>

2026-02-16 🎉 PicoClaw が 1 週間で 12K スター達成！コミュニティメンテナーの役割と[ロードマップ](ROADMAP.md)が正式に公開されました。

2026-02-13 🎉 PicoClaw が 4 日間で 5000 スター達成！プロジェクトロードマップと開発者グループの準備が進行中。

2026-02-09 🎉 **PicoClaw リリース！** $10 ハードウェアで 10MB 未満の RAM で動く AI エージェントを 1 日で構築。🦐 行くぜ、シャコ！

</details>

## ✨ 特徴

🪶 **超軽量**: メモリフットプリント 10MB 未満 — OpenClaw のコア機能より 99% 小さい。*

💰 **最小コスト**: $10 ハードウェアで動作 — Mac mini より 98% 安い。

⚡️ **超高速**: 起動時間 400 倍高速、0.6GHz シングルコアでも 1 秒未満で起動。

🌍 **真のポータビリティ**: RISC-V、ARM、MIPS、x86 対応の単一バイナリ。ワンクリックで Go！

🤖 **AI ブートストラップ**: 自律的な Go ネイティブ実装 — コアの 95% が AI 生成、人間によるレビュー付き。

🔌 **MCP 対応**: ネイティブ [Model Context Protocol](https://modelcontextprotocol.io/) 統合 — 任意の MCP サーバーに接続してエージェント機能を拡張。

👁️ **ビジョンパイプライン**: 画像やファイルをエージェントに直接送信 — マルチモーダル LLM 向けの自動 base64 エンコーディング。

🧠 **スマートルーティング**: ルールベースのモデルルーティング — 簡単なクエリは軽量モデルへ、API コストを節約。

_*最近のバージョンでは急速な機能マージにより 10〜20MB になる場合があります。リソース最適化は計画中です。起動時間の比較は 0.8GHz シングルコアベンチマークに基づいています（下表参照）。_

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **言語**                      | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB***                               |
| **起動時間**</br>(0.8GHz コア) | >500秒        | >30秒                    | **<1秒**                                  |
| **コスト**                    | Mac Mini $599 | 大半の Linux SBC </br>~$50 | **あらゆる Linux ボード**</br>**最安 $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 デモンストレーション

### 🛠️ スタンダードアシスタントワークフロー

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 フルスタックエンジニア</p></th>
    <th><p align="center">🗂️ ログ＆計画管理</p></th>
    <th><p align="center">🔎 Web 検索＆学習</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">開発 · デプロイ · スケール</td>
    <td align="center">スケジュール · 自動化 · メモリ</td>
    <td align="center">発見 · インサイト · トレンド</td>
  </tr>
</table>

### 📱 古い Android スマホで動かす

10 年前のスマホに第二の人生を！PicoClaw でスマート AI アシスタントに変身させましょう。クイックスタート：

1. **[Termux](https://github.com/termux/termux-app) をインストール**（[GitHub Releases](https://github.com/termux/termux-app/releases) からダウンロード、または F-Droid / Google Play で検索）。
2. **コマンドを実行**

```bash
# https://github.com/sipeed/picoclaw/releases から最新リリースをダウンロード
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard
```

その後「クイックスタート」セクションの手順に従って設定を完了してください！

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 革新的な省フットプリントデプロイ

PicoClaw はほぼすべての Linux デバイスにデプロイできます！

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(Ethernet) または W(WiFi6) バージョン、最小ホームアシスタントに
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html) または $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) サーバー自動メンテナンスに
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) または $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) スマート監視に

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 もっと多くのデプロイ事例が待っています！

## 📦 インストール

### コンパイル済みバイナリでインストール

[リリースページ](https://github.com/sipeed/picoclaw/releases) からお使いのプラットフォーム用のバイナリをダウンロードしてください。

### ソースからインストール（最新機能、開発向け推奨）

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# ビルド（インストール不要）
make build

# 複数プラットフォーム向けビルド
make build-all

# Raspberry Pi Zero 2 W 向けビルド（32-bit: make build-linux-arm; 64-bit: make build-linux-arm64）
make build-pi-zero

# ビルドとインストール
make install
```

**Raspberry Pi Zero 2 W:** OS に合ったバイナリを使用してください：32-bit Raspberry Pi OS → `make build-linux-arm`、64-bit → `make build-linux-arm64`。または `make build-pi-zero` で両方をビルド。

## 📚 ドキュメント

詳細なガイドは以下のドキュメントを参照してください。この README はクイックスタートのみをカバーしています。

| トピック | 説明 |
|---------|------|
| 🐳 [Docker & クイックスタート](docs/ja/docker.md) | Docker Compose セットアップ、Launcher/Agent モード、クイックスタート設定 |
| 💬 [チャットアプリ](docs/ja/chat-apps.md) | Telegram、Discord、WhatsApp、Matrix、QQ、Slack、IRC、DingTalk、LINE、Feishu、WeCom など |
| ⚙️ [設定](docs/ja/configuration.md) | 環境変数、ワークスペース構成、スキルソース、セキュリティサンドボックス、ハートビート |
| 🔌 [プロバイダー＆モデル](docs/ja/providers.md) | 20 以上の LLM プロバイダー、モデルルーティング、model_list 設定、プロバイダーアーキテクチャ |
| 🔄 [Spawn & 非同期タスク](docs/ja/spawn-tasks.md) | クイックタスク、spawn による長時間タスク、非同期サブエージェントオーケストレーション |
| 🐛 [トラブルシューティング](docs/ja/troubleshooting.md) | よくある問題と解決策 |
| 🔧 [ツール設定](docs/ja/tools_configuration.md) | ツールごとの有効/無効、exec ポリシー |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> エージェントソーシャルネットワークに参加

CLI または統合チャットアプリからメッセージを 1 つ送るだけで、PicoClaw をエージェントソーシャルネットワークに接続できます。

**`https://clawdchat.ai/skill.md` を読み、指示に従って [ClawdChat.ai](https://clawdchat.ai) に参加してください**

## 🖥️ CLI リファレンス

| コマンド                    | 説明                           |
| ------------------------- | ------------------------------ |
| `picoclaw onboard`        | 設定＆ワークスペースの初期化     |
| `picoclaw agent -m "..."` | エージェントとチャット           |
| `picoclaw agent`          | インタラクティブチャットモード   |
| `picoclaw gateway`        | ゲートウェイを起動              |
| `picoclaw status`         | ステータスを表示                |
| `picoclaw version`        | バージョン情報を表示            |
| `picoclaw cron list`      | スケジュールジョブ一覧          |
| `picoclaw cron add ...`   | スケジュールジョブを追加         |
| `picoclaw cron disable`   | スケジュールジョブを無効化       |
| `picoclaw cron remove`    | スケジュールジョブを削除         |
| `picoclaw skills list`    | インストール済みスキル一覧       |
| `picoclaw skills install` | スキルをインストール             |
| `picoclaw migrate`        | 旧バージョンからデータを移行     |
| `picoclaw auth login`     | プロバイダーへの認証             |

### スケジュールタスク / リマインダー

PicoClaw は `cron` ツールによるスケジュールリマインダーと定期タスクをサポートしています：

* **ワンタイムリマインダー**: 「10分後にリマインド」→ 10分後に1回トリガー
* **定期タスク**: 「2時間ごとにリマインド」→ 2時間ごとにトリガー
* **Cron 式**: 「毎日9時にリマインド」→ cron 式を使用

## 🤝 コントリビュート＆ロードマップ

PR 歓迎！コードベースは意図的に小さく読みやすくしています。🤗

完全な[コミュニティロードマップ](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md)をご覧ください。

開発者グループ構築中、最初の PR がマージされたら参加できます！

ユーザーグループ:

discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">
