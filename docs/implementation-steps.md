# Dual-Flavor 実装ステップ

設計書: `docs/android-dual-flavor-plan.md` に基づく段階的実装計画。
各ステップは独立してビルド検証可能な単位に分割している。

---

## Config をアプリから設定できるようにする

### Step 1: Go — Gateway HTTP サーバー + Config API エンドポイント

Gateway HTTP サーバーを新規作成し、Config API を実装する。

**新規 or 修正ファイル（Go 側）:**
- Gateway HTTP サーバー作成（port 18790）
  - `GET /api/config` — 設定値返却（api_key はマスク）
  - `GET /api/config/schema` — スキーマ返却（フィールド名、型、ラベル、デフォルト値、セクション階層）
  - `PUT /api/config` — 設定保存 → サーバー再起動トリガー
  - HTTP API 認証:
    - `Authorization: Bearer <api_key>` で検証
    - `api_key` 未設定時は認証スキップ

**検証:**
- `go test ./...` パス
- `curl http://127.0.0.1:18790/api/config/schema` でスキーマ取得
- `curl http://127.0.0.1:18790/api/config` で設定取得
- `curl -X PUT http://127.0.0.1:18790/api/config -d '{...}'` で設定保存
- `api_key` 設定時: Bearer なし/不一致で HTTP API が拒否される
- `api_key` 設定時: `Authorization: Bearer <api_key>` で HTTP API が通る

---

### Step 2: :backend:api モジュール — 骨格 + インターフェース/モデル

**作成ファイル:**
- `android/backend/api/build.gradle.kts` — android-library, coroutines 依存のみ
- `android/backend/api/src/main/AndroidManifest.xml`
- `BackendState.kt`
- `BackendLifecycle.kt`
- `NoopBackendLifecycle.kt`
- `GatewaySettings.kt`
- `GatewaySettingsStore.kt`

**修正ファイル:**
- `android/settings.gradle.kts` — `include(":backend:api")` 追加

**検証:** `./gradlew :backend:api:compileDebugKotlin`

---

### Step 3: :backend:config モジュール + ConfigApiClient

**作成ファイル:**
- `android/backend/config/build.gradle.kts` — android-library, compose, navigation, ktor-client, `:backend:api`, `:core:ui`（テーマ共有）
- `android/backend/config/src/main/AndroidManifest.xml`
- `ConfigApiClient.kt`

**修正ファイル:**
- `android/settings.gradle.kts` — `include(":backend:config")` 追加
- `android/app/build.gradle.kts` — `implementation(project(":backend:api"))`, `implementation(project(":backend:config"))` 追加

**検証:** `./gradlew :backend:config:compileDebugKotlin`

---

### Step 4: Config UI（ViewModel + Screen）

**作成ファイル:**
- `ConfigViewModel.kt`
- `ConfigSectionListScreen.kt`
- `ConfigSectionDetailScreen.kt`
- `ConfigModule.kt`

**検証:** `./gradlew :backend:config:compileDebugKotlin`

---

### Step 5: SAF ワークスペース（Config UI に統合）

**修正ファイル:**
- `ConfigSectionDetailScreen.kt`:
  - ワークスペースフィールドに SAF ピッカーボタン追加
  - `ACTION_OPEN_DOCUMENT_TREE` + `takePersistableUriPermission`
  - URI → ファイルパス変換（内部ストレージのみ対応）
  - 非対応パス選択時のエラー表示 + フォールバック

**検証:**
- ワークスペースを任意ディレクトリに変更可能
- 非対応パス（SDカード等）選択時にエラー表示

---

### Step 6: AppSettingsScreen + ナビゲーション統合

**作成ファイル:**
- `app/src/main/java/io/clawdroid/settings/AppSettingsScreen.kt`

**修正ファイル:**
- `NavRoutes.kt` — `BACKEND_SETTINGS`, `BACKEND_SETTINGS_SECTION`, `APP_SETTINGS` 追加
- `MainActivity.kt` — ルート追加
- `ClawDroidApp.kt` — `configModule` 追加

**検証:** Termux 環境で設定画面 → セクション一覧 → 詳細 → 値編集 → 保存

---

## 接続設定 + API キー

### Step 7: GatewaySettingsStore + WebSocketClient apiKey + ConfigApiClient 連携

**作成ファイル:**
- `app/src/main/java/io/clawdroid/settings/GatewaySettingsStoreImpl.kt`

**修正ファイル:**
- `AppModule.kt` — `GatewaySettingsStore` 登録 + WebSocketClient に settings 反映
- `WebSocketClient.kt` — `var apiKey` 追加、`connect()` で `&api_key=` 付加
- `ClawDroidApp.kt` — settings observe → WS 再接続
- `ConfigApiClient.kt` — コンストラクタに `GatewaySettingsStore` 追加、リクエストごとに `settings.value` からポート + API キーを取得
- `ConfigModule.kt` — `single { ConfigApiClient(get()) }` に更新（GatewaySettingsStore 注入）
- `AppSettingsScreen.kt` — 保存ボタンの `GatewaySettingsStore.update()` 呼び出しを有効化

**検証:** ビルド成功、接続設定変更で WS 再接続、AppSettings で保存 → ConfigApiClient が新ポート/キーを使用

---

### Step 8: Go — WS API キー認証 + setup_required メッセージ

**修正ファイル:**
- `pkg/channels/websocket.go`:
  - WS upgrade 時に `?api_key=` クエリパラメータで認証
  - 未設定時は認証スキップ
  - LLM API key 未設定等で `{"type": "setup_required", "content": "..."}` 送信
- `MainActivity.kt` — `setup_required` observe → 設定画面遷移

**検証:**
- API key 不一致で WS 接続拒否
- LLM 未設定で setup_required → 設定画面に自動遷移

---

## Flavor 分離

### Step 9: Product Flavors + :backend:loader-noop + FlavorModule

**作成ファイル:**
- `android/backend/loader-noop/build.gradle.kts`
- `android/backend/loader-noop/src/main/AndroidManifest.xml`
- `app/src/termux/java/io/clawdroid/di/FlavorModule.kt`
- `app/src/embedded/java/io/clawdroid/di/FlavorModule.kt`（初期は Noop）

**修正ファイル:**
- `android/settings.gradle.kts` — `include(":backend:loader-noop")` 追加
- `app/build.gradle.kts` — flavors, buildConfig, sourceSets, flavor 依存追加
- `ClawDroidApp.kt` — `modules(appModule, flavorModule, configModule)`

**検証:** `./gradlew assembleTermuxDebug` と `./gradlew assembleEmbeddedDebug` 両方成功

---

## Embedded 固有

### Step 10: Makefile build-android

**修正ファイル:**
- `Makefile` — `build-android` ターゲット追加（arm64, x86_64, armv7）

**検証:** `make build-android` → `android/app/src/embedded/jniLibs/` に .so 生成

---

### Step 11: :backend:loader + GatewayProcessManager

**作成ファイル:**
- `android/backend/loader/build.gradle.kts`
- `android/backend/loader/src/main/AndroidManifest.xml` — ForegroundService 宣言
- `GatewayProcessManager.kt`

**修正ファイル:**
- `android/settings.gradle.kts` — `include(":backend:loader")` 追加
- `android/app/build.gradle.kts` — `embeddedImplementation(project(":backend:loader"))` 追加

**検証:** `./gradlew :backend:loader:compileDebugKotlin`

---

### Step 12: EmbeddedBackendLifecycle + GatewayService

**作成ファイル:**
- `EmbeddedBackendLifecycle.kt`
- `GatewayService.kt` — LifecycleService, ForegroundService, START_STICKY

**検証:** `./gradlew :backend:loader:compileDebugKotlin`

---

### Step 13: embedded FlavorModule + 起動フロー統合

**修正ファイル:**
- `app/src/embedded/java/io/clawdroid/di/FlavorModule.kt` — EmbeddedBackendLifecycle に差し替え
- `ClawDroidApp.kt` — `BackendLifecycle.start()` 呼び出し追加

**検証:**
- `./gradlew assembleEmbeddedDebug` ビルド成功
- embedded 版で Go プロセス起動 → WS 接続 → チャット動作
- `adb shell ps | grep clawdroid` でプロセス確認
