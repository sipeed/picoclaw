# PicoClaw Android APK - チャットUI設計プラン

## Context

PicoClawはTermux上で動作するGo製AIエージェント。WebSocketサーバーチャネル(`pkg/channels/websocket.go`)を通じて外部クライアントと通信可能。このWSサーバーに接続するAndroid APKを作成する。初期リリースはテキスト+画像チャットUI、将来的にボイスUI（Googleアシスタント風）を追加予定。

## WSプロトコル（確認済み）

```
Endpoint: ws://127.0.0.1:18793/ws

Client → Server:  {"content":"text", "sender_id":"optional", "images":["raw_base64_no_prefix"]}
Server → Client:  {"content":"text"}
```

- imagesはraw base64（`data:`プレフィックスなし）。サーバー側で`data:image/png;base64,`を付与（websocket.go:237）
- sender_id未指定時、サーバーがUUIDを割り当て（websocket.go:227-230）

---

## 技術スタック

| ツール | バージョン | 備考 |
|--------|-----------|------|
| AGP | **9.0.1** | Kotlin組み込み、`kotlin-android`プラグイン不要 |
| Gradle | **9.1+** | AGP 9.0要件 |
| Kotlin | **2.3.0** (AGP組み込み) | |
| Compose BOM | **2026.01.01** | |
| Ktor Client | **3.4.0** | OkHttpエンジン + WebSocketプラグイン |
| kotlinx.serialization | **1.10.0** | コンパイラプラグイン |
| Koin | **4.1.1** | DSLベースDI |
| Room | **2.8.4** | SQLite永続化（KSP必要） |
| KSP | **2.3.6** | Room用 |
| compileSdk / targetSdk | **36** | |
| minSdk | **28** | |

---

## アーキテクチャ — マルチモジュール構成

**Jetpack Compose + MVVM + Clean Architecture + Feature Module分離**

```
:app                  → アプリのエントリーポイント、DI、ナビゲーション
:feature:chat         → チャットUI（Compose画面、ViewModel、UIコンポーネント）
:core:domain          → ドメインモデル、リポジトリinterface、UseCases
:core:data            → リポジトリ実装、Room DB、WebSocketClient、DTO
:core:ui              → 共有テーマ、共有コンポーネント
```

**依存関係**:
```
:app → :feature:chat, :core:domain, :core:data, :core:ui
:feature:chat → :core:domain, :core:ui
:core:data → :core:domain
:core:ui → (compose dependencies only)
```

**利点**: ボイスUI追加時は `:feature:voice` を新設するだけ。`:core:domain`と`:core:data`はそのまま共有。

---

## ディレクトリ構成

```
android/
├── .gitignore
├── build.gradle.kts                          # ルートビルド
├── settings.gradle.kts                       # 全モジュール include
├── gradle.properties
├── gradle/
│   └── libs.versions.toml
│
├── app/                                       # === :app モジュール ===
│   ├── build.gradle.kts
│   ├── proguard-rules.pro
│   └── src/main/
│       ├── AndroidManifest.xml
│       ├── res/
│       │   ├── xml/network_security_config.xml
│       │   ├── values/strings.xml
│       │   ├── values/colors.xml
│       │   └── values/themes.xml
│       └── java/io/picoclaw/android/
│           ├── PicoClawApp.kt                 # Application（Koin初期化）
│           ├── MainActivity.kt                 # Single Activity + Navigation
│           └── di/
│               └── AppModule.kt               # Koin全体module定義
│
├── feature/
│   └── chat/                                  # === :feature:chat モジュール ===
│       ├── build.gradle.kts
│       └── src/main/java/io/picoclaw/android/feature/chat/
│           ├── ChatViewModel.kt
│           ├── ChatUiState.kt
│           ├── ChatEvent.kt
│           ├── screen/
│           │   └── ChatScreen.kt
│           └── component/
│               ├── MessageBubble.kt
│               ├── MessageInput.kt
│               ├── ConnectionBanner.kt
│               ├── ImagePreview.kt
│               └── MessageList.kt
│
├── core/
│   ├── domain/                                # === :core:domain モジュール ===
│   │   ├── build.gradle.kts
│   │   └── src/main/java/io/picoclaw/android/core/domain/
│   │       ├── model/
│   │       │   ├── ChatMessage.kt
│   │       │   ├── ConnectionState.kt
│   │       │   └── ImageAttachment.kt
│   │       ├── repository/
│   │       │   └── ChatRepository.kt          # interface
│   │       └── usecase/
│   │           ├── SendMessageUseCase.kt
│   │           ├── ObserveMessagesUseCase.kt
│   │           ├── ObserveConnectionUseCase.kt
│   │           └── LoadMoreMessagesUseCase.kt
│   │
│   ├── data/                                  # === :core:data モジュール ===
│   │   ├── build.gradle.kts
│   │   └── src/main/java/io/picoclaw/android/core/data/
│   │       ├── remote/
│   │       │   ├── WebSocketClient.kt         # Ktor WS + auto-reconnect
│   │       │   └── dto/
│   │       │       ├── WsIncoming.kt          # @Serializable
│   │       │       └── WsOutgoing.kt          # @Serializable
│   │       ├── local/
│   │       │   ├── AppDatabase.kt             # Room Database
│   │       │   ├── entity/
│   │       │   │   └── MessageEntity.kt       # Room Entity
│   │       │   ├── dao/
│   │       │   │   └── MessageDao.kt          # Room DAO
│   │       │   └── converter/
│   │       │       └── Converters.kt          # TypeConverter
│   │       ├── mapper/
│   │       │   └── MessageMapper.kt           # Entity ↔ Domain, DTO ↔ Entity
│   │       └── repository/
│   │           └── ChatRepositoryImpl.kt
│   │
│   └── ui/                                    # === :core:ui モジュール ===
│       ├── build.gradle.kts
│       └── src/main/java/io/picoclaw/android/core/ui/
│           └── theme/
│               ├── Theme.kt
│               ├── Color.kt
│               └── Type.kt
```

---

## 各層の設計

### :core:data — Data層

#### Room DB（メッセージ永続化）

**MessageEntity**:
```kotlin
@Entity(tableName = "messages")
data class MessageEntity(
    @PrimaryKey val id: String,          // UUID
    val content: String,
    val sender: String,                   // "USER" or "AGENT"
    val imageBase64List: String?,         // JSON文字列 ["base64_1","base64_2"]
    val timestamp: Long,                  // epoch millis
    val status: String                    // "SENDING","SENT","FAILED","RECEIVED"
)
```

**MessageDao**:
```kotlin
@Dao
interface MessageDao {
    @Query("SELECT * FROM messages ORDER BY timestamp DESC LIMIT :limit")
    fun getRecentMessages(limit: Int): Flow<List<MessageEntity>>

    @Query("SELECT * FROM messages WHERE timestamp < :beforeTimestamp ORDER BY timestamp DESC LIMIT :limit")
    suspend fun getMessagesBefore(beforeTimestamp: Long, limit: Int): List<MessageEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(message: MessageEntity)

    @Update
    suspend fun update(message: MessageEntity)
}
```

**AppDatabase**:
```kotlin
@Database(entities = [MessageEntity::class], version = 1)
@TypeConverters(Converters::class)
abstract class AppDatabase : RoomDatabase() {
    abstract fun messageDao(): MessageDao
}
```

#### ページネーション戦略

```
起動時:
  → DAO.getRecentMessages(limit=50) をFlowでobserve
  → 最新50件が表示される → 新メッセージはinsert → Flowが自動更新

スクロール遡り:
  → LazyColumnの先頭付近に到達を検知
  → Repository.loadMore() → _displayLimitを+30
  → RoomのFlowが自動的に再クエリ → UIに反映
```

#### WebSocketClient — Ktor Client (OkHttpエンジン) ベース
- `connectionState: StateFlow<ConnectionState>` で接続状態を公開
- `incomingMessages: SharedFlow<WsOutgoing>` でサーバー応答を公開
- `connect()` / `disconnect()` / `send(dto)` メソッド
- exponential backoff自動再接続（1s→2s→4s→8s→max 30s）

```kotlin
val client = HttpClient(OkHttp) {
    install(WebSockets)
    engine {
        preconfigured = OkHttpClient.Builder()
            .pingInterval(30, TimeUnit.SECONDS)
            .build()
    }
}
```

#### ChatRepositoryImpl — Room + WS統合

```kotlin
class ChatRepositoryImpl(
    private val webSocketClient: WebSocketClient,
    private val messageDao: MessageDao,
    private val scope: CoroutineScope
) : ChatRepository {

    private val _displayLimit = MutableStateFlow(INITIAL_LOAD_COUNT) // 50

    override val messages: StateFlow<List<ChatMessage>> =
        _displayLimit.flatMapLatest { limit ->
            messageDao.getRecentMessages(limit)
        }.map { entities ->
            entities.map { MessageMapper.toDomain(it) }.reversed()
        }.stateIn(scope, SharingStarted.Lazily, emptyList())

    override val connectionState = webSocketClient.connectionState

    init {
        scope.launch {
            webSocketClient.incomingMessages.collect { dto ->
                val entity = MessageMapper.toEntity(dto)
                messageDao.insert(entity)
            }
        }
    }

    override suspend fun sendMessage(text: String, images: List<ImageAttachment>) {
        val entity = MessageMapper.toEntity(text, images, MessageStatus.SENDING)
        messageDao.insert(entity)
        val wsDto = MessageMapper.toWsIncoming(text, images)
        val success = webSocketClient.send(wsDto)
        messageDao.update(entity.copy(status = if (success) "SENT" else "FAILED"))
    }

    override fun loadMore() {
        _displayLimit.update { it + PAGE_SIZE }
    }

    companion object {
        const val INITIAL_LOAD_COUNT = 50
        const val PAGE_SIZE = 30
    }
}
```

#### DTO — `@Serializable`

```kotlin
@Serializable
data class WsIncoming(
    val content: String,
    @SerialName("sender_id") val senderId: String? = null,
    val images: List<String>? = null
)

@Serializable
data class WsOutgoing(val content: String)
```

### :core:domain — Domain層

**ChatMessage** — `id, content, sender(USER/AGENT), images: List<String>, timestamp, status`

**ConnectionState** — `DISCONNECTED, CONNECTING, CONNECTED, RECONNECTING`

**ImageAttachment** — `uri, base64, mimeType`

**ChatRepository interface**:
```kotlin
interface ChatRepository {
    val messages: StateFlow<List<ChatMessage>>
    val connectionState: StateFlow<ConnectionState>
    suspend fun sendMessage(text: String, images: List<ImageAttachment> = emptyList())
    fun loadMore()
    fun connect()
    fun disconnect()
}
```

**UseCases** — `SendMessageUseCase`, `ObserveMessagesUseCase`, `ObserveConnectionUseCase`, `LoadMoreMessagesUseCase`

### :core:ui — 共有UI

テーマ定義（Color, Type, Theme）。将来的に共有コンポーネントもここに配置。

### :feature:chat — チャットUI

**ChatUiState**:
```kotlin
data class ChatUiState(
    val messages: List<ChatMessage> = emptyList(),
    val connectionState: ConnectionState = ConnectionState.DISCONNECTED,
    val inputText: String = "",
    val pendingImages: List<ImageAttachment> = emptyList(),
    val isLoadingMore: Boolean = false,
    val canLoadMore: Boolean = true,
    val error: String? = null
)
```

**ChatViewModel** — Koin `koinViewModel()`で注入

**Compose UI階層**:
```
ChatScreen
├── ConnectionBanner          # 接続状態（赤/黄/非表示）
├── MessageList (weight 1f)   # LazyColumn (reverseLayout=true)
│   ├── Loading indicator     # isLoadingMore時
│   └── MessageBubble         # 右=ユーザー、左=エージェント
└── Bottom Column
    ├── ImagePreviewRow       # 送信前サムネイル
    └── MessageInput          # [カメラ][ギャラリー][TextField][送信]
```

**スクロール検知でloadMore**:
```kotlin
val listState = rememberLazyListState()
val shouldLoadMore by remember {
    derivedStateOf {
        val lastVisibleItem = listState.layoutInfo.visibleItemsInfo.lastOrNull()
        lastVisibleItem != null &&
            lastVisibleItem.index >= listState.layoutInfo.totalItemsCount - 5
    }
}
LaunchedEffect(shouldLoadMore) {
    if (shouldLoadMore) viewModel.onLoadMore()
}
```

---

## DI（Koin）

```kotlin
// app/di/AppModule.kt
val appModule = module {
    // Room
    single {
        Room.databaseBuilder(androidContext(), AppDatabase::class.java, "picoclaw.db").build()
    }
    single { get<AppDatabase>().messageDao() }

    // Ktor HttpClient
    single {
        HttpClient(OkHttp) {
            install(WebSockets)
            engine {
                preconfigured = OkHttpClient.Builder()
                    .pingInterval(30, TimeUnit.SECONDS)
                    .build()
            }
        }
    }

    // WebSocketClient
    single { WebSocketClient(get()) }

    // Repository
    single<ChatRepository> { ChatRepositoryImpl(get(), get(), get()) }

    // UseCases
    factory { SendMessageUseCase(get()) }
    factory { ObserveMessagesUseCase(get()) }
    factory { ObserveConnectionUseCase(get()) }
    factory { LoadMoreMessagesUseCase(get()) }

    // ViewModel
    viewModel { ChatViewModel(get(), get(), get(), get(), get()) }
}
```

---

## Gradle設定

### settings.gradle.kts
```kotlin
pluginManagement {
    repositories { google(); mavenCentral(); gradlePluginPortal() }
}
dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories { google(); mavenCentral() }
}
rootProject.name = "PicoClaw"
include(":app")
include(":feature:chat")
include(":core:domain")
include(":core:data")
include(":core:ui")
```

### gradle/libs.versions.toml
```toml
[versions]
agp = "9.0.1"
kotlin = "2.3.0"
ksp = "2.3.6"
compose-bom = "2026.01.01"
ktor = "3.4.0"
koin = "4.1.1"
serialization = "1.10.0"
room = "2.8.4"
lifecycle = "2.10.0"
activity-compose = "1.12.4"
core-ktx = "1.17.0"
coroutines = "1.10.2"

[libraries]
# Compose
compose-bom = { group = "androidx.compose", name = "compose-bom", version.ref = "compose-bom" }
compose-ui = { group = "androidx.compose.ui", name = "ui" }
compose-ui-tooling = { group = "androidx.compose.ui", name = "ui-tooling" }
compose-ui-tooling-preview = { group = "androidx.compose.ui", name = "ui-tooling-preview" }
compose-material3 = { group = "androidx.compose.material3", name = "material3" }
compose-icons-extended = { group = "androidx.compose.material", name = "material-icons-extended" }

# AndroidX
activity-compose = { group = "androidx.activity", name = "activity-compose", version.ref = "activity-compose" }
core-ktx = { group = "androidx.core", name = "core-ktx", version.ref = "core-ktx" }
lifecycle-runtime-compose = { group = "androidx.lifecycle", name = "lifecycle-runtime-compose", version.ref = "lifecycle" }
lifecycle-viewmodel-compose = { group = "androidx.lifecycle", name = "lifecycle-viewmodel-compose", version.ref = "lifecycle" }

# Ktor
ktor-client-okhttp = { group = "io.ktor", name = "ktor-client-okhttp", version.ref = "ktor" }
ktor-client-websockets = { group = "io.ktor", name = "ktor-client-websockets", version.ref = "ktor" }
ktor-client-content-negotiation = { group = "io.ktor", name = "ktor-client-content-negotiation", version.ref = "ktor" }
ktor-serialization-json = { group = "io.ktor", name = "ktor-serialization-kotlinx-json", version.ref = "ktor" }

# Serialization
serialization-json = { group = "org.jetbrains.kotlinx", name = "kotlinx-serialization-json", version.ref = "serialization" }

# Room
room-runtime = { group = "androidx.room", name = "room-runtime", version.ref = "room" }
room-ktx = { group = "androidx.room", name = "room-ktx", version.ref = "room" }
room-compiler = { group = "androidx.room", name = "room-compiler", version.ref = "room" }

# Koin
koin-android = { group = "io.insert-koin", name = "koin-android", version.ref = "koin" }
koin-compose = { group = "io.insert-koin", name = "koin-androidx-compose", version.ref = "koin" }

# Coroutines
coroutines-android = { group = "org.jetbrains.kotlinx", name = "kotlinx-coroutines-android", version.ref = "coroutines" }

[plugins]
android-application = { id = "com.android.application", version.ref = "agp" }
android-library = { id = "com.android.library", version.ref = "agp" }
serialization = { id = "org.jetbrains.kotlin.plugin.serialization", version.ref = "kotlin" }
ksp = { id = "com.google.devtools.ksp", version.ref = "ksp" }
```

### 各モジュールのbuild.gradle.kts概要

**:app** — `android-application`, `serialization` → depends on all modules
**:feature:chat** — `android-library` → depends on `:core:domain`, `:core:ui`
**:core:domain** — `android-library` → depends on nothing (pure Kotlin + coroutines)
**:core:data** — `android-library`, `serialization`, `ksp` → depends on `:core:domain` (Room, Ktor, DTO)
**:core:ui** — `android-library` → depends on nothing (Compose theme only)

KSPは`:core:data`モジュールのみで使用（Room compiler）。

---

## 注意点

- **Cleartext通信**: `ws://127.0.0.1`への通信に`network_security_config.xml`でlocalhost/127.0.0.1のみ許可
- **画像base64**: サーバーが`data:image/png;base64,`を自動付与するため、クライアントはraw base64のみ送信
- **画像のDB保存**: 画像base64はMessageEntityにJSON配列文字列として保存。TypeConverterで変換
- **Ktor WS ping**: OkHttpエンジン使用時はOkHttpBuilder側の`pingInterval`で設定
- **AGP 9.0**: `kotlin-android`プラグイン不適用、`kotlin.compilerOptions`を使用
- **マルチモジュール**: 各library moduleは`android.namespace`を個別設定

---

## 実装順序

### Phase 1: プロジェクトスキャフォールド + マルチモジュール
1. `android/` ディレクトリ構造（5モジュール分）
2. ルート `build.gradle.kts`、`settings.gradle.kts`（全module include）、`libs.versions.toml`
3. `gradle.properties`
4. 各モジュールの `build.gradle.kts`
5. `app/` の `AndroidManifest.xml` + `network_security_config.xml` + リソース
6. Gradleラッパー生成

### Phase 2: :core:domain
7. `ConnectionState.kt`、`ChatMessage.kt`、`ImageAttachment.kt`
8. `ChatRepository.kt` (interface)
9. 4つのUseCase

### Phase 3: :core:ui
10. Theme（Color.kt, Type.kt, Theme.kt）

### Phase 4: :core:data — Room
11. `MessageEntity.kt` + `Converters.kt`
12. `MessageDao.kt`
13. `AppDatabase.kt`

### Phase 5: :core:data — WebSocket + Repository
14. `WsIncoming.kt`、`WsOutgoing.kt` (@Serializable)
15. `MessageMapper.kt`
16. `WebSocketClient.kt`（Ktor + auto-reconnect）
17. `ChatRepositoryImpl.kt`（Room + WS統合、ページネーション）

### Phase 6: :feature:chat
18. `ChatUiState.kt`、`ChatEvent.kt`
19. `ChatViewModel.kt`（loadMore対応）
20. UIコンポーネント（ConnectionBanner, MessageBubble, MessageList, ImagePreview, MessageInput）
21. `ChatScreen.kt`（スクロール検知+ページネーション）

### Phase 7: :app
22. `AppModule.kt`（Koin全体module）
23. `PicoClawApp.kt`
24. `MainActivity.kt`

---

## 検証方法

1. Termuxでpicoclaw起動（WSサーバーが`127.0.0.1:18793`でリスン）
2. APKをビルド・インストール（`./gradlew assembleDebug`）
3. アプリ起動 → ConnectionBannerが「Connected」表示を確認
4. テキスト送信 → エージェントからの応答がチャットに表示されることを確認
5. アプリを再起動 → 前回のチャット履歴がRoomから読み込まれて表示されることを確認
6. 50件以上メッセージを蓄積 → 上にスクロール → 古いメッセージが動的にロードされることを確認
7. カメラ/ギャラリーから画像添付+テキスト送信 → エージェントが画像を認識した応答を返すことを確認
8. WSサーバーを停止 → 「Reconnecting」表示 → サーバー再起動後に自動再接続を確認

---

## 将来の拡張

- **:feature:voice**: `:core:domain`と`:core:data`をそのまま共有。SpeechRecognizer+TextToSpeechのUI層のみ新設
- **複数サーバー**: `WebSocketClient`のurl引数を動的に
- **メッセージ検索**: Room DAOにFTS4クエリ追加

---

## 参照するサーバー側ファイル

| ファイル | 用途 |
|---------|------|
| `pkg/channels/websocket.go` | WSプロトコル定義 |
| `pkg/bus/types.go` | InboundMessage/OutboundMessage構造体 |
| `pkg/channels/base.go` | Channelインターフェース |
| `pkg/config/config.go` | WebSocketConfig |
