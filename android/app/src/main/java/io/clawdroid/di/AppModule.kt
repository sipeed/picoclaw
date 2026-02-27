package io.clawdroid.di

import androidx.room.Room
import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.websocket.WebSockets
import io.clawdroid.core.data.local.AppDatabase
import io.clawdroid.assistant.AccessibilityScreenshotSource
import io.clawdroid.assistant.DeviceController
import io.clawdroid.core.data.local.ImageFileStorage
import io.clawdroid.feature.chat.voice.ScreenshotSource
import io.clawdroid.core.data.remote.WebSocketClient
import io.clawdroid.assistant.ToolRequestHandler
import io.clawdroid.core.data.repository.ChatRepositoryImpl
import io.clawdroid.core.data.repository.TtsCatalogRepositoryImpl
import io.clawdroid.core.data.repository.TtsSettingsRepositoryImpl
import io.clawdroid.core.domain.repository.ChatRepository
import io.clawdroid.core.domain.repository.TtsCatalogRepository
import io.clawdroid.core.domain.repository.TtsSettingsRepository
import io.clawdroid.core.domain.usecase.ConnectChatUseCase
import io.clawdroid.core.domain.usecase.DisconnectChatUseCase
import io.clawdroid.core.domain.usecase.LoadMoreMessagesUseCase
import io.clawdroid.core.domain.usecase.ObserveConnectionUseCase
import io.clawdroid.core.domain.usecase.ObserveMessagesUseCase
import io.clawdroid.core.domain.usecase.ObserveStatusUseCase
import io.clawdroid.core.domain.usecase.SendMessageUseCase
import io.clawdroid.backend.api.GatewaySettingsStore
import io.clawdroid.feature.chat.ChatViewModel
import io.clawdroid.feature.chat.SettingsViewModel
import io.clawdroid.feature.chat.voice.SpeechRecognizerWrapper
import io.clawdroid.feature.chat.voice.TextToSpeechWrapper
import io.clawdroid.feature.chat.voice.CameraCaptureManager
import io.clawdroid.feature.chat.voice.VoiceModeManager
import io.clawdroid.settings.AppSettingsViewModel
import io.clawdroid.settings.GatewaySettingsStoreImpl
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import okhttp3.OkHttpClient
import org.koin.android.ext.koin.androidContext
import org.koin.core.module.dsl.viewModel
import org.koin.dsl.module
import java.util.UUID
import java.util.concurrent.TimeUnit

val appModule = module {
    // CoroutineScope
    single { CoroutineScope(SupervisorJob() + Dispatchers.IO) }

    // GatewaySettingsStore
    single<GatewaySettingsStore> { GatewaySettingsStoreImpl(androidContext(), get()) }

    // Room
    single {
        Room.databaseBuilder(
            androidContext(),
            AppDatabase::class.java,
            "clawdroid.db"
        ).fallbackToDestructiveMigration(dropAllTables = true).build()
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
    single {
        val prefs = androidContext().getSharedPreferences("clawdroid", android.content.Context.MODE_PRIVATE)
        val clientId = prefs.getString("client_id", null) ?: UUID.randomUUID().toString().also {
            prefs.edit().putString("client_id", it).apply()
        }
        WebSocketClient(get(), get(), clientId)
    }

    // ImageFileStorage
    single { ImageFileStorage(androidContext()) }

    // Repository
    single<ChatRepository> {
        val repo = ChatRepositoryImpl(get(), get(), get(), get())
        val handler = ToolRequestHandler(
            context = androidContext(),
            deviceController = get(),
            screenshotSource = get(),
            setOverlayVisibility = {},
            onAccessibilityNeeded = {}
        )
        repo.onToolRequest = { request ->
            val response = handler.handle(request)
            if (response.success) response.result ?: "" else response.error ?: "unknown error"
        }
        repo
    }
    single<TtsSettingsRepository> { TtsSettingsRepositoryImpl(androidContext()) }
    single<TtsCatalogRepository> {
        TtsCatalogRepositoryImpl(androidContext(), get<TtsSettingsRepository>().ttsConfig, get())
    }

    // UseCases
    factory { SendMessageUseCase(get()) }
    factory { ObserveMessagesUseCase(get()) }
    factory { ObserveConnectionUseCase(get()) }
    factory { ObserveStatusUseCase(get()) }
    factory { LoadMoreMessagesUseCase(get()) }
    factory { ConnectChatUseCase(get()) }
    factory { DisconnectChatUseCase(get()) }

    // Screenshot
    single { AccessibilityScreenshotSource() }
    single<ScreenshotSource> { get<AccessibilityScreenshotSource>() }

    // Device Controller (for Android tool)
    single { DeviceController() }

    // Voice
    factory { SpeechRecognizerWrapper(androidContext()) }
    single { TextToSpeechWrapper(androidContext(), get<TtsSettingsRepository>().ttsConfig) }
    single { CameraCaptureManager(androidContext()) }
    single { VoiceModeManager(get(), get(), get(), get(), get(), get()) }

    // ViewModel
    viewModel { ChatViewModel(get(), get(), get(), get(), get(), get(), get(), get()) }
    viewModel { SettingsViewModel(get(), get(), get()) }
    viewModel { AppSettingsViewModel(get(), get()) }
}
