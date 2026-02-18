package io.picoclaw.android.di

import androidx.room.Room
import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.websocket.WebSockets
import io.picoclaw.android.core.data.local.AppDatabase
import io.picoclaw.android.core.data.local.ImageFileStorage
import io.picoclaw.android.core.data.remote.WebSocketClient
import io.picoclaw.android.core.data.repository.ChatRepositoryImpl
import io.picoclaw.android.core.domain.repository.ChatRepository
import io.picoclaw.android.core.domain.usecase.LoadMoreMessagesUseCase
import io.picoclaw.android.core.domain.usecase.ObserveConnectionUseCase
import io.picoclaw.android.core.domain.usecase.ObserveMessagesUseCase
import io.picoclaw.android.core.domain.usecase.SendMessageUseCase
import io.picoclaw.android.feature.chat.ChatViewModel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import okhttp3.OkHttpClient
import org.koin.android.ext.koin.androidContext
import org.koin.core.module.dsl.viewModel
import org.koin.dsl.module
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
        ).build()
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

    // ImageFileStorage
    single { ImageFileStorage(androidContext()) }

    // Repository
    single<ChatRepository> { ChatRepositoryImpl(get(), get(), get(), get()) }

    // UseCases
    factory { SendMessageUseCase(get()) }
    factory { ObserveMessagesUseCase(get()) }
    factory { ObserveConnectionUseCase(get()) }
    factory { LoadMoreMessagesUseCase(get()) }

    // ViewModel
    viewModel { ChatViewModel(get(), get(), get(), get(), get()) }
}
