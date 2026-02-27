# ClawDroid Dual-Flavor APK 実装計画

## Context

ClawDroidは現在、Termux上でGoバイナリを手動起動し、Android APK（Kotlin/Compose）がWebSocketで接続する構成。技術的に詳しくないユーザーにも使えるよう、Goバイナリを内蔵したスタンドアロン版APKを追加する。Gradle Product Flavorsで2つのバリアントを管理する。

## 2つのFlavor

| | termux | embedded |
|---|---|---|
| applicationId | `io.clawdroid`（変更なし） | `io.clawdroid`（変更なし） |
| バックエンド | ユーザーがTermuxで起動 | APK内蔵バイナリを自動起動 |
| 設定 | Config API経由でアプリ内UIから管理 | Config API経由でアプリ内UIから管理 |
| ワークスペース | SAFでユーザー選択（termuxで直接変更も可） | SAFでユーザー選択 |
| Gateway API キー | アプリ設定で手動入力（サーバー側はユーザーが管理） | 初回サービス起動時に自動生成（更新時はサービス経由で同期） |
| ポート | アプリ設定でデフォルト指定（変更可） | サービス経由で管理（デフォルト使用、アプリ設定から変更可） |

※ applicationId は同一。同一デバイスには片方のみインストール可能。APKファイル名で区別。

---

## モジュール構成

機能分離は **独立したGradleモジュール** で行い、flavor source sets は DI 配線と embedded バイナリ配置に限定する。

```
android/
├── app/                        # :app — エントリー、DI、ナビゲーション
│   └── src/
│       ├── main/               # 共通コード（既存）
│       ├── termux/java/.../di/FlavorModule.kt    # DI配線のみ
│       └── embedded/
│           ├── java/.../di/FlavorModule.kt        # DI配線のみ
│           └── jniLibs/{abi}/libclawdroid.so      # Goバイナリ（embedded限定）
│
├── backend/
│   ├── api/                    # :backend:api — interface + モデル + NoopBackendLifecycle（共通）
│   ├── loader/                 # :backend:loader — プロセス管理（embedded用）
│   ├── loader-noop/            # :backend:loader-noop — 空実装（termux用、re-export のみ）
│   └── config/                 # :backend:config — Config APIクライアント + 設定UI
│
├── feature/chat/               # :feature:chat（既存、変更なし）
├── core/domain/                # :core:domain（既存、変更なし）
├── core/data/                  # :core:data（WebSocketClient 接続パラメータ injectable 化のみ）
└── core/ui/                    # :core:ui（既存、変更なし）
```

### 依存関係

```
:app
  ├── (共通) :feature:chat, :core:domain, :core:data, :core:ui, :backend:api, :backend:config
  ├── (termux) :backend:loader-noop
  └── (embedded) :backend:loader

:backend:api         → coroutines のみ（純Kotlin）— NoopBackendLifecycle のデフォルト実装を含む
:backend:loader      → :backend:api
:backend:loader-noop → :backend:api（re-export のみ）
:backend:config      → :backend:api, :core:ui（テーマ共有）

:feature:chat        → :core:domain, :core:ui（変更なし）
```

---

## Phase 1: :backend:config モジュール — Config API + 自動生成UI

### 設計方針

Go側にConfig APIを追加し、Androidはconfig.goの構造を一切知らない。
APIからスキーマとデータを取得し、Compose UIを動的に生成する。
実装順は `implementation-steps.md` に従い、`:backend:api`（Step 2）を先行した上で進める。

### Go側の変更（config API追加）

**修正**: `pkg/channels/websocket.go` or 新規 API ハンドラ

Gateway HTTP サーバー（port 18790）に以下エンドポイント追加:

- `GET /api/config` → 現在の設定値をJSON返却（api_keyはマスク）
- `GET /api/config/schema` → スキーマ情報返却（フィールド名、型、ラベル、デフォルト値、セクション階層）
- `PUT /api/config` → 設定を保存 → サーバー再起動

API キー認証（Go 側で2箇所のバリデーションが必要）:
- **HTTP API**: リクエストヘッダー `Authorization: Bearer <api_key>` で検証
- **WebSocket**: upgrade リクエストのクエリパラメータ `?api_key=...` で検証（WS upgrade 時はカスタムヘッダーが使いにくいため）
- 未設定時（api_key 空）は両方とも認証スキップ

スキーマ例:
```json
{
  "sections": [
    {
      "key": "llm", "label": "LLM",
      "fields": [
        {"key": "model", "type": "string", "label": "Model", "default": ""},
        {"key": "api_key", "type": "string", "label": "API Key", "secret": true},
        {"key": "base_url", "type": "string", "label": "Base URL"}
      ]
    },
    ...
  ]
}
```

WS接続時、設定未完了（LLM API key未設定等）の場合:
```json
{"type": "setup_required", "content": "LLM API key is not configured"}
```
→ Android側はこのメッセージを受けて設定画面に自動遷移

### 1-1. :backend:config モジュール

**新規**: `backend/config/build.gradle.kts` — android-library, compose, navigation, ktor-client

### 1-2. Config API クライアント

**新規**: `backend/config/src/main/java/io/clawdroid/backend/config/ConfigApiClient.kt`
- Ktor HTTP client で Gateway API にアクセス
- Step 3-6 では `127.0.0.1:18790`（api_key なし）を使用
- Step 7 以降、接続先ポート + API キーは **リクエストごとに** `GatewaySettingsStore.settings.value` から取得（キャッシュしない）
  - ポート変更時に次回リクエストから自動的に新ポートを使用
- `suspend fun getSchema(): ConfigSchema`
- `suspend fun getConfig(): Map<String, Any?>`
- `suspend fun saveConfig(config: Map<String, Any?>): Result` → サーバー再起動トリガー

### 1-3. 動的 Compose UI

**新規**: `backend/config/src/main/java/io/clawdroid/backend/config/ConfigSectionListScreen.kt`
- 設定画面を開く → ローディングスピナー → API からスキーマ+設定値を取得 → 描画
- トップレベル: 各セクションをカード表示、タップで詳細へ

**新規**: `backend/config/src/main/java/io/clawdroid/backend/config/ConfigSectionDetailScreen.kt`
- APIスキーマの型情報に応じて描画:
  - `string` → TextField（`secret: true` なら PasswordTextField）
  - `bool` → Switch
  - `int` → TextField with number keyboard
  - `float` → TextField with decimal keyboard
- サブセクション → ネストしたナビゲーション
- ワークスペースフィールド → SAFピッカーボタン付き（内部ストレージのみ対応の注記表示）

**新規**: `backend/config/src/main/java/io/clawdroid/backend/config/ConfigViewModel.kt`
- `ConfigApiClient` を注入
- `StateFlow<ConfigUiState>` (Loading / Loaded / Saving / Saved / Reconnecting / Error)
- 保存ボタン押下 → Saving 表示 → `saveConfig()` → 「設定を保存しました。再接続中...」表示 → WS再接続完了で自動復帰

### 1-4. setup_required ハンドリング（実装順は Step 8）

Go が WS 接続時に `{"type": "setup_required"}` を送信した場合、アプリは設定画面に自動遷移する。

実装方針（`:core:data` を変更しない — 接続パラメータ変更は Step 7 で実施）:
- `WebSocketClient` は `AppModule.kt` で `single` 登録済み → `:app` から Koin 経由で直接取得可能
- `MainActivity` で `WebSocketClient.incomingMessages` を observe し、
  `setup_required` タイプを検知してナビゲーションイベントを発火:
  ```kotlin
  val wsClient: WebSocketClient by inject()
  wsClient.incomingMessages.collect { message ->
      if (message.type == "setup_required") {
          navController.navigate(NavRoutes.BACKEND_SETTINGS)
      }
  }
  ```

### 1-5. ナビゲーション統合

**修正**: `app/src/main/java/io/clawdroid/navigation/NavRoutes.kt`
```kotlin
object NavRoutes {
    const val CHAT = "chat"
    const val SETTINGS = "settings"
    const val BACKEND_SETTINGS = "backend_settings"
    const val BACKEND_SETTINGS_SECTION = "backend_settings/{sectionKey}"
    const val APP_SETTINGS = "app_settings"
}
```

**修正**: `app/src/main/java/io/clawdroid/MainActivity.kt`
- backend_settings ルートを追加（両flavor共通 — Config APIはGo側なのでtermuxでもアクセス可能）
- アプリ設定（Gateway 接続設定）への導線を追加
- SettingsScreen から Backend Settings への導線を MainActivity 側で制御
  （`:feature:chat` の SettingsScreen は変更しない）

### 1-6. アプリ設定画面（Gateway 接続設定）

**新規**: `app/src/main/java/io/clawdroid/settings/AppSettingsScreen.kt`
- 実装順: Step 6 で画面とナビゲーション導線を実装。Gateway 接続設定（WS/HTTP ポート、API キー）の反映処理は Step 7 で有効化
- Gateway API キー: TextField（PasswordTextField）
  - embedded: 自動生成済みの値が表示される。変更するとサービス経由でサーバーも更新
  - termux: ユーザーが手動入力。アプリ内のキーのみ更新
- Gateway WS ポート: TextField with number keyboard（デフォルト: 18793）
- Gateway HTTP ポート: TextField with number keyboard（デフォルト: 18790）
- 保存（Step 7） → `GatewaySettingsStore.update()` → WebSocketClient が自動再接続

### 1-7. Config モジュール内に Koin モジュール定義

**新規**: `backend/config/src/main/java/io/clawdroid/backend/config/ConfigModule.kt`
```kotlin
val configModule = module {
    single { ConfigApiClient() }  // Step 7 で GatewaySettingsStore 連携版に更新
    viewModel { ConfigViewModel(get()) }
}
```

**修正**: `app/src/main/java/io/clawdroid/ClawDroidApp.kt`
```kotlin
modules(appModule, configModule)  // Step 6: configModule 追加
```
※ `flavorModule` は Step 9 で追加する。

### 検証
- Step 6 時点（flavor導入前）は termux 構成で設定画面導線（Backend/App Settings）と `設定画面 → セクション一覧 → 詳細 → 値編集 → 保存` を確認
- API key入力 → 保存 → 「保存しました。再接続中...」表示 → プロセス再起動 → チャット動作
- Step 7 で Gateway 接続設定の変更 → 即座に再接続
- 両flavor での同等動作確認は Step 9 以降のビルドで検証

---

## Phase 2: SAFワークスペース

### 2-1. SAFディレクトリ選択

ConfigSectionDetailScreen のワークスペースフィールドに SAF ピッカーを統合:
- `ACTION_OPEN_DOCUMENT_TREE` で任意ディレクトリ選択
- `takePersistableUriPermission` で永続アクセス権取得
- URI → ファイルパス変換（`/storage/emulated/0/...` 配下のみ対応）
- SDカード・USB OTG: 現時点では非対応。UI 上で「内部ストレージのみ対応」と明示
- 非対応パス選択時: エラー表示 + デフォルト（`getExternalFilesDir("workspace")`）にフォールバック
- デフォルト: `getExternalFilesDir("workspace")`

### 検証
- ワークスペースを Downloads に設定 → ファイルマネージャーで確認
- Goバイナリがそのディレクトリに読み書きできること
- 非対応パス（SDカード等）選択時にエラー表示されること

### 注: feature/chat, core/domain, core/ui モジュールは変更なし

WebSocketクライアントの既存の再接続ロジック（CONNECTING/RECONNECTING表示）が
バックエンド起動待ちを暗黙的にカバーするため、ChatUiState/ConnectionBannerの修正は不要。
Backend Settings へのナビゲーションは `MainActivity.kt` で両flavor共通の
トップレベルルートとして追加する。

---

## Phase 3: Gradle Flavor + :backend:loader-noop（接続設定は Step 7 で実装）

### 3-1. `settings.gradle.kts` にモジュール追加（Step 9 対応分）

```kotlin
include(":backend:loader-noop")
```

※ `:backend:api` は Step 2、`:backend:config` は Step 3 で追加済み。  
※ `:backend:loader` は Step 11 で追加する。

### 3-2. `app/build.gradle.kts` に flavors + flavor依存

```kotlin
android {
    flavorDimensions += "variant"
    productFlavors {
        create("termux") { dimension = "variant" }
        create("embedded") { dimension = "variant" }
    }
    buildFeatures {
        compose = true
        buildConfig = true  // BuildConfig.FLAVOR で判定
    }
    sourceSets {
        getByName("termux") { java.srcDirs("src/termux/java") }
        getByName("embedded") {
            java.srcDirs("src/embedded/java")
        }
    }
}

dependencies {
    // 共通
    implementation(project(":backend:api"))
    implementation(project(":backend:config"))
    // ... 既存依存 ...

    // flavor固有
    "termuxImplementation"(project(":backend:loader-noop"))
}
```

※ `embeddedImplementation(project(":backend:loader"))` は Step 11 で追加する。

### 3-3. :backend:api モジュール（実装順は Step 2 で先行）

**新規**: `backend/api/build.gradle.kts` — android-library, coroutines依存のみ

**新規**: `backend/api/src/main/java/io/clawdroid/backend/api/BackendState.kt`
```kotlin
enum class BackendState { STOPPED, STARTING, RUNNING, ERROR }
```

**新規**: `backend/api/src/main/java/io/clawdroid/backend/api/BackendLifecycle.kt`
```kotlin
interface BackendLifecycle {
    val state: StateFlow<BackendState>
    val isManaged: Boolean  // false for termux, true for embedded
    suspend fun start()
    suspend fun stop()
}
```

**新規**: `backend/api/src/main/java/io/clawdroid/backend/api/NoopBackendLifecycle.kt`
```kotlin
/** デフォルト実装 — バックエンドは常に RUNNING（termux用、embedded 初期スタブ兼用） */
class NoopBackendLifecycle : BackendLifecycle {
    override val state = MutableStateFlow(BackendState.RUNNING)
    override val isManaged = false
    override suspend fun start() {}
    override suspend fun stop() {}
}
```

### 3-4. :backend:loader-noop モジュール（termux用）

**新規**: `backend/loader-noop/build.gradle.kts` — android-library, `:backend:api` に依存
- `NoopBackendLifecycle` は `:backend:api` にあるため、このモジュールは依存の分離のみ担当
- 将来 termux 固有ロジック（Termux:API intent 連携等）が必要になった場合の拡張ポイント

### 3-5. Gateway 接続設定（アプリローカル設定、実装順は Step 7）

Gateway API キーとポートは **Config API とは別のアプリローカル設定**（SharedPreferences / DataStore）で管理する。
WebSocketClient と ConfigApiClient の接続先を動的に設定可能にする。
Step 7 で `ConfigApiClient` を `GatewaySettingsStore` 参照版に更新する。

**新規**: `backend/api/src/main/java/io/clawdroid/backend/api/GatewaySettings.kt`
```kotlin
/** Gateway サーバーへの接続設定 */
data class GatewaySettings(
    val wsPort: Int = 18793,
    val httpPort: Int = 18790,
    val apiKey: String = "",  // 空 = 認証なし
)
```

**新規**: `backend/api/src/main/java/io/clawdroid/backend/api/GatewaySettingsStore.kt`
```kotlin
/** アプリローカルの Gateway 接続設定ストア */
interface GatewaySettingsStore {
    val settings: StateFlow<GatewaySettings>
    suspend fun update(settings: GatewaySettings)
}
```

**新規**: `app/src/main/java/io/clawdroid/settings/GatewaySettingsStoreImpl.kt`
```kotlin
/** DataStore ベースの実装（プロジェクトの datastore-preferences に準拠） */
class GatewaySettingsStoreImpl(context: Context) : GatewaySettingsStore {
    // DataStore<Preferences> で wsPort, httpPort, apiKey を永続化
    // StateFlow で変更を公開
}
```

**修正**: `core/data/src/main/java/io/clawdroid/core/data/remote/WebSocketClient.kt`
- 現在 `var wsUrl: String = "ws://127.0.0.1:18793/ws"` がパブリック var として存在
- コンストラクタ変更なし（`HttpClient, CoroutineScope, clientId, clientType`）
- `connect()` 内で URL 構築時に API キーをクエリパラメータに追加:
  ```kotlin
  val url = "$wsUrl?client_id=$clientId&client_type=$clientType&api_key=$apiKey"
  ```
- `apiKey` プロパティを追加（`var apiKey: String = ""`）
- wsUrl / apiKey の変更は外部（AppModule の settings observe）で行い、
  変更時に `disconnect()` → `connect()` で再接続

### 3-6. Flavor source sets（DI配線のみ）

**新規**: `app/src/termux/java/io/clawdroid/di/FlavorModule.kt`
```kotlin
val flavorModule = module {
    single<BackendLifecycle> { NoopBackendLifecycle() }
}
```

**新規**: `app/src/embedded/java/io/clawdroid/di/FlavorModule.kt`
```kotlin
val flavorModule = module {
    // Phase 4 で EmbeddedBackendLifecycle に差し替え。Phase 3 では NoopBackendLifecycle をスタブ使用
    single<BackendLifecycle> { NoopBackendLifecycle() }
}
```

### 3-7. 既存ファイル修正

**修正**: `app/src/main/java/io/clawdroid/ClawDroidApp.kt`
```kotlin
modules(appModule, flavorModule, configModule)  // flavorModule 追加（configModule は Step 6 で追加済み）
```

Koin 初期化後、GatewaySettingsStore の変更を observe して WebSocketClient を再接続:
```kotlin
val settingsStore: GatewaySettingsStore = get()
val wsClient: WebSocketClient = get()
scope.launch {
    settingsStore.settings
        .drop(1)  // 初回値スキップ（AppModule で初期化済み）
        .collect { s ->
            wsClient.wsUrl = "ws://127.0.0.1:${s.wsPort}/ws"
            wsClient.apiKey = s.apiKey
            wsClient.disconnect()
            wsClient.connect()
        }
}
```

**修正**: `app/src/main/java/io/clawdroid/di/AppModule.kt`
```kotlin
// GatewaySettingsStore
single<GatewaySettingsStore> { GatewaySettingsStoreImpl(androidContext()) }

// WebSocketClient — コンストラクタは変更なし、wsUrl と apiKey を settings から設定
single {
    val prefs = androidContext().getSharedPreferences("clawdroid", android.content.Context.MODE_PRIVATE)
    val clientId = prefs.getString("client_id", null) ?: UUID.randomUUID().toString().also {
        prefs.edit().putString("client_id", it).apply()
    }
    val settingsStore = get<GatewaySettingsStore>()
    val settings = settingsStore.settings.value
    WebSocketClient(get(), get(), clientId).apply {
        wsUrl = "ws://127.0.0.1:${settings.wsPort}/ws"
        apiKey = settings.apiKey
    }
}

```

### 検証
- `./gradlew assembleTermuxDebug` と `./gradlew assembleEmbeddedDebug` 両方ビルド成功
- termux版は現状と同じ動作（デフォルトポートで接続）

---

## Phase 4: :backend:loader モジュール — バイナリ同梱 + プロセス管理

### バイナリ配置方式: jniLibs アプローチ（embedded source set）

Go バイナリを `libclawdroid.so` としてネイティブライブラリに偽装し、**embedded flavor の `jniLibs`** に配置する。
termux 版 APK にはバイナリが含まれない。

これにより:
- Android 標準の ABI フィルタリングが自動で効く（ABI split 不要）
- `context.applicationInfo.nativeLibraryDir` から直接実行可能
- `filesDir` へのコピーや実行権限付与が不要
- SELinux の W^X ポリシー制限を回避（nativeLibraryDir は exec 許可済み）
- termux 版 APK サイズに影響なし

### 4-1. Makefile に `build-android` ターゲット追加

**修正**: `Makefile`
```makefile
build-android: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath $(LDFLAGS) \
		-o android/app/src/embedded/jniLibs/arm64-v8a/libclawdroid.so ./cmd/clawdroid
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath $(LDFLAGS) \
		-o android/app/src/embedded/jniLibs/x86_64/libclawdroid.so ./cmd/clawdroid
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath $(LDFLAGS) \
		-o android/app/src/embedded/jniLibs/armeabi-v7a/libclawdroid.so ./cmd/clawdroid
```

注: x86（32bit）は非対応。エミュレータ開発時は x86_64 を使用すること。

### 4-2. :backend:loader モジュール

**新規**: `backend/loader/build.gradle.kts` — android-library, lifecycle-service, coroutines

**修正**: `settings.gradle.kts`（Step 11）
```kotlin
include(":backend:loader")
```

**修正**: `app/build.gradle.kts`（Step 11）
```kotlin
"embeddedImplementation"(project(":backend:loader"))
```

**新規**: `backend/loader/src/main/java/io/clawdroid/backend/loader/GatewayProcessManager.kt`
- `ProcessBuilder` で `nativeLibraryDir/libclawdroid.so gateway run` を起動
  - バイナリパス: `context.applicationInfo.nativeLibraryDir + "/libclawdroid.so"`
- 環境変数:
  - `HOME` → `context.filesDir`（Go側の `~/.clawdroid/config.json` がアプリ内に解決される）
  - `CLAWDROID_GATEWAY_API_KEY` → GatewaySettingsStore から取得した API キー
  - 他の設定はGoが `config.json` から読み取る。変更は Config API (`PUT /api/config`) 経由
- stdout/stderr → `Log.i` に転送（別スレッドで読み取り）
- プロセス死亡時 exponential backoff で再起動（1s→2s→4s→...→30s上限）
- `StateFlow<BackendState>` で状態公開
- WebSocket接続可能までポーリング → `RUNNING` 遷移
- **PID ファイル管理**:
  - 起動時: PID を `filesDir/clawdroid.pid` に記録
  - 起動前: PID ファイルが存在すれば `/proc/<pid>/cmdline` でプロセス生存確認 → 生存なら kill
  - 停止時・onDestroy: PID ファイル削除
- **API キー初回生成** (embedded):
  - GatewaySettingsStore の apiKey が空なら `UUID.randomUUID()` で生成
  - 生成したキーを GatewaySettingsStore に保存 + Go プロセスに環境変数で渡す
- **API キー更新**:
  - GatewaySettingsStore の変更を observe → Go プロセスを新しいキーで再起動

**新規**: `backend/loader/src/main/java/io/clawdroid/backend/loader/EmbeddedBackendLifecycle.kt`
- `BackendLifecycle` 実装
- `start()` = GatewayService が起動済みか判定 → 未起動なら ForegroundService を起動
- `isManaged = true`
- `state` は GatewayProcessManager の StateFlow を委譲

**新規**: `backend/loader/src/main/java/io/clawdroid/backend/loader/GatewayService.kt`
- `LifecycleService` 継承の ForegroundService
- **Go プロセスのオーナー** — Service が起動/停止の責任を持つ
- `onCreate`:
  - `GatewayProcessManager.start()` — 孤児チェック → API キー初回生成 → Go プロセス起動
  - `startForeground()` で通知表示（"ClawDroid バックエンド実行中"）
- `onStartCommand`:
  ```kotlin
  override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
      // intent が null でも問題なし — 初期化は onCreate で完了済み
      // START_STICKY による OS kill 後の再起動時は intent=null で呼ばれる
      return START_STICKY
  }
  ```
- `onDestroy` → `GatewayProcessManager.stop()`
- アプリがバックグラウンドに行ってもServiceが生き続けるのでGoプロセスも継続

**新規**: `backend/loader/src/main/AndroidManifest.xml`
```xml
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE_SPECIAL_USE" />

    <application>
        <service
            android:name=".loader.GatewayService"
            android:exported="false"
            android:foregroundServiceType="specialUse">
            <property
                android:name="android.app.PROPERTY_SPECIAL_USE_FGS_SUBTYPE"
                android:value="Local AI agent backend process" />
        </service>
    </application>
</manifest>
```

注: minSdk 30 のため `FOREGROUND_SERVICE` は必ずサポート済み。
`FOREGROUND_SERVICE_SPECIAL_USE` は API 34+ で必須だが、API 30-33 では属性が無視されるだけで問題なし。

### 4-3. FlavorModule 更新（embedded）

**修正**: `app/src/embedded/java/io/clawdroid/di/FlavorModule.kt`
```kotlin
val flavorModule = module {
    single { GatewayProcessManager(androidContext(), get()) }  // GatewaySettingsStore を注入
    single<BackendLifecycle> { EmbeddedBackendLifecycle(androidContext(), get()) }
}
```

### 4-4. アプリ起動フロー

**修正**: `app/src/main/java/io/clawdroid/ClawDroidApp.kt`
- Koin初期化後に `BackendLifecycle.start()` を呼ぶ
  - **termux版**: no-op（Termux側でGoが起動済み前提）
  - **embedded版**: GatewayServiceが起動済みか判定
    - 起動済み → 何もしない（WS接続はWebSocketClientの自動再接続に任せる）
    - 未起動 → `startForegroundService(Intent(GatewayService))` で起動
    - Service内: API キー初回生成（未設定時） → Go プロセス起動 → `libclawdroid.so gateway run`

### 検証
- embedded版でアプリ起動 → API キー自動生成 → Goプロセスが起動 → WebSocket接続成功 → チャットが動作
- `adb shell ps | grep clawdroid` でプロセス確認
- termux版は影響なし

---

## 新規ファイル一覧

| ファイル | Step | モジュール |
|---------|------|-----------|
| `backend/config/build.gradle.kts` | 3 | :backend:config |
| `backend/config/.../ConfigApiClient.kt` | 3 | :backend:config |
| `backend/config/.../ConfigSectionListScreen.kt` | 4 | :backend:config |
| `backend/config/.../ConfigSectionDetailScreen.kt` | 4 | :backend:config |
| `backend/config/.../ConfigViewModel.kt` | 4 | :backend:config |
| `backend/config/.../ConfigModule.kt` | 4 | :backend:config |
| `app/src/main/.../settings/AppSettingsScreen.kt` | 6 | :app |
| Go: config API handler（新規 or 既存ファイル修正） | 1 | Go側 |
| `backend/api/build.gradle.kts` | 2 | :backend:api |
| `backend/api/.../BackendState.kt` | 2 | :backend:api |
| `backend/api/.../BackendLifecycle.kt` | 2 | :backend:api |
| `backend/api/.../NoopBackendLifecycle.kt` | 2 | :backend:api |
| `backend/api/.../GatewaySettings.kt` | 2 | :backend:api |
| `backend/api/.../GatewaySettingsStore.kt` | 2 | :backend:api |
| `backend/loader-noop/build.gradle.kts` | 9 | :backend:loader-noop |
| `backend/loader-noop/src/main/AndroidManifest.xml` | 9 | :backend:loader-noop |
| `app/src/main/.../settings/GatewaySettingsStoreImpl.kt` | 7 | :app |
| `app/src/termux/java/.../di/FlavorModule.kt` | 9 | :app (flavor) |
| `app/src/embedded/java/.../di/FlavorModule.kt` | 9-13 | :app (flavor) |
| `app/src/embedded/jniLibs/{abi}/libclawdroid.so` | 10 | :app (jniLibs, embedded限定) |
| `backend/loader/build.gradle.kts` | 11 | :backend:loader |
| `backend/loader/.../GatewayProcessManager.kt` | 11 | :backend:loader |
| `backend/loader/.../EmbeddedBackendLifecycle.kt` | 12 | :backend:loader |
| `backend/loader/.../GatewayService.kt` | 12 | :backend:loader |
| `backend/loader/src/main/AndroidManifest.xml` | 11 | :backend:loader |

## 修正ファイル一覧

| ファイル | Step | 変更内容 |
|---------|------|---------|
| `android/app/.../NavRoutes.kt` | 6 | `BACKEND_SETTINGS`, `BACKEND_SETTINGS_SECTION`, `APP_SETTINGS` 追加 |
| `android/app/.../MainActivity.kt` | 6, 8 | 設定画面ナビゲーション追加、`setup_required` observe 追加 |
| `android/app/.../ClawDroidApp.kt` | 6, 7, 9, 13 | `configModule`/`flavorModule` 追加、settings observe、`BackendLifecycle.start()` |
| `android/settings.gradle.kts` | 2, 3, 9, 11 | `:backend:api`/`:backend:config`/`:backend:loader-noop`/`:backend:loader` を段階的に include |
| `android/app/build.gradle.kts` | 3, 9, 11 | `:backend:config` 依存、flavors/sourceSets/loader-noop 依存、`embeddedImplementation(:backend:loader)` |
| `android/app/.../di/AppModule.kt` | 7 | `GatewaySettingsStore` 登録、WebSocketClient 注入変更 |
| `core/data/.../WebSocketClient.kt` | 7 | `var apiKey: String` 追加、`connect()` で API キーをクエリパラメータに付加（コンストラクタ変更なし） |
| `backend/config/.../ConfigSectionDetailScreen.kt` | 5 | SAF ワークスペース選択統合 |
| `pkg/channels/websocket.go` | 1, 8 | Config API 関連処理、WS `api_key` 認証と `setup_required` 送信 |
| `Makefile` | 10 | `build-android` ターゲット追加 |

**変更なし**: `:feature:chat`, `:core:domain`, `:core:ui`
**最小限の変更**: `:core:data` — WebSocketClient の接続パラメータ injectable 化のみ

---

## リスクと対策

1. **jniLibs exec 互換性** — `nativeLibraryDir` からの exec は全 Android バージョンで許可（標準的な NDK ライブラリ配置）。Go バイナリは ELF 形式で `.so` 拡張子でも正常動作。万が一問題があれば `filesDir` コピー+exec にフォールバック
2. **APKサイズ** — Goバイナリ ~30MB/ABI。embedded source set の jniLibs に配置するため、termux 版 APK には含まれない。Android 標準の ABI フィルタリングにより、デバイスには該当 ABI のみインストール。Play Store 配信時は App Bundle で自動分割
3. **SAFパス変換** — content URI → ファイルパスは `/storage/emulated/0/` 配下のみ対応。SDカード・USB OTG は非対応とし、UI 上で「内部ストレージのみ対応」と明示。非対応パス選択時は `getExternalFilesDir` にフォールバック
4. **プロセス孤児化** — `GatewayProcessManager` が PID を `filesDir/clawdroid.pid` に記録。起動時に `/proc/<pid>/cmdline` で孤児チェック → 生存なら kill → PID ファイル更新。`onDestroy` で PID ファイル削除
5. **config.go との同期** — API化により不要。Go側にフィールド追加すればスキーマAPIが自動反映
6. **同一applicationId** — 両flavorが同ID、同一端末に共存不可。APKファイル名で区別
7. **Config 保存後の再接続** — `PUT /api/config` 後にGoプロセス再起動でWS切断。ConfigViewModel が UI 状態を Saving → Saved → Reconnecting と遷移させ、既存の WebSocket 再接続ロジック（exponential backoff）で自動復旧
8. **ForegroundService パーミッション** — `FOREGROUND_SERVICE`（minSdk 30 で必ずサポート）と `FOREGROUND_SERVICE_SPECIAL_USE`（API 34+）を AndroidManifest に宣言。API 30-33 では specialUse 属性が無視されるだけで問題なし
9. **x86 ABI** — 非対応。エミュレータ開発時は x86_64 を使用。必要になれば `GOARCH=386` ビルド追加で対応可
10. **Gateway API キー セキュリティ** — localhost 通信のためネットワーク経由の盗聴リスクは低い。主な目的は同一端末上の他アプリからの不正アクセス防止
11. **ProGuard / R8** — `:backend:config` は Ktor HTTP クライアントで JSON を動的処理するため、release ビルド時に R8 でシリアライゼーション関連クラスが strip されないよう ProGuard ルール追加が必要。既存の `proguard-rules.pro` を確認し、Ktor + kotlinx.serialization のルールを追記
