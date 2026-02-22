package io.picoclaw.android.di

import androidx.room.Room
import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.websocket.WebSockets
import io.picoclaw.android.core.data.local.AppDatabase
import io.picoclaw.android.assistant.AccessibilityScreenshotSource
import io.picoclaw.android.assistant.DeviceController
import io.picoclaw.android.core.data.local.ImageFileStorage
import io.picoclaw.android.feature.chat.voice.ScreenshotSource
import io.picoclaw.android.core.data.remote.WebSocketClient
import io.picoclaw.android.assistant.ToolRequestHandler
import io.picoclaw.android.core.data.repository.ChatRepositoryImpl
import io.picoclaw.android.core.data.repository.TtsCatalogRepositoryImpl
import io.picoclaw.android.core.data.repository.TtsSettingsRepositoryImpl
import io.picoclaw.android.core.domain.repository.ChatRepository
import io.picoclaw.android.core.domain.repository.TtsCatalogRepository
import io.picoclaw.android.core.domain.repository.TtsSettingsRepository
import io.picoclaw.android.core.domain.usecase.ConnectChatUseCase
import io.picoclaw.android.core.domain.usecase.DisconnectChatUseCase
import io.picoclaw.android.core.domain.usecase.LoadMoreMessagesUseCase
import io.picoclaw.android.core.domain.usecase.ObserveConnectionUseCase
import io.picoclaw.android.core.domain.usecase.ObserveMessagesUseCase
import io.picoclaw.android.core.domain.usecase.ObserveStatusUseCase
import io.picoclaw.android.core.domain.usecase.SendMessageUseCase
import io.picoclaw.android.feature.chat.ChatViewModel
import io.picoclaw.android.feature.chat.SettingsViewModel
import io.picoclaw.android.feature.chat.voice.SpeechRecognizerWrapper
import io.picoclaw.android.feature.chat.voice.TextToSpeechWrapper
import io.picoclaw.android.feature.chat.voice.CameraCaptureManager
import io.picoclaw.android.feature.chat.voice.VoiceModeManager
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

    // Room
    single {
        Room.databaseBuilder(
            androidContext(),
            AppDatabase::class.java,
            "picoclaw.db"
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
        val prefs = androidContext().getSharedPreferences("picoclaw", android.content.Context.MODE_PRIVATE)
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
}
