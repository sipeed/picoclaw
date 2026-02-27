package io.clawdroid

import android.app.Application
import android.util.Log
import io.clawdroid.backend.api.GatewaySettingsStore
import io.clawdroid.backend.config.ConfigApiClient
import io.clawdroid.backend.config.configModule
import io.clawdroid.core.data.remote.WebSocketClient
import io.clawdroid.di.appModule
import io.clawdroid.receiver.NotificationHelper
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.koin.android.ext.koin.androidContext
import org.koin.core.context.startKoin
import java.net.URLEncoder

class ClawDroidApp : Application() {
    override fun onCreate() {
        super.onCreate()
        val koinApp = startKoin {
            androidContext(this@ClawDroidApp)
            modules(appModule, configModule)
        }
        NotificationHelper.createNotificationChannel(this)

        val koin = koinApp.koin
        val settingsStore: GatewaySettingsStore = koin.get()
        val wsClient: WebSocketClient = koin.get()
        val configApiClient: ConfigApiClient = koin.get()
        val scope: CoroutineScope = koin.get()
        scope.launch {
            // Only react when httpPort or apiKey actually change
            settingsStore.settings
                .map { it.httpPort to it.apiKey }
                .distinctUntilChanged()
                .collect {
                    // Fetch WS connection info from config API
                    val wsUrl = fetchWsUrl(configApiClient)
                    if (wsClient.wsUrl != wsUrl) {
                        wsClient.disconnect()
                        wsClient.wsUrl = wsUrl
                        wsClient.connect()
                    }
                }
        }
    }

    private suspend fun fetchWsUrl(configApiClient: ConfigApiClient): String {
        return try {
            val cfg = configApiClient.getConfig()
            val ws = cfg["channels"]?.jsonObject?.get("websocket")?.jsonObject
            if (ws != null) {
                val host = ws["host"]?.jsonPrimitive?.content ?: "127.0.0.1"
                val port = ws["port"]?.jsonPrimitive?.content ?: "18793"
                val path = ws["path"]?.jsonPrimitive?.content ?: "/ws"
                val wsApiKey = ws["api_key"]?.jsonPrimitive?.content ?: ""
                buildString {
                    append("ws://$host:$port$path")
                    if (wsApiKey.isNotEmpty()) {
                        append("?api_key=${URLEncoder.encode(wsApiKey, "UTF-8")}")
                    }
                }
            } else {
                DEFAULT_WS_URL
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to fetch WS config from API, using defaults", e)
            DEFAULT_WS_URL
        }
    }

    companion object {
        private const val TAG = "ClawDroidApp"
        private const val DEFAULT_WS_URL = "ws://127.0.0.1:18793/ws"
    }
}
