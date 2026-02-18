package io.picoclaw.android.core.data.remote

import android.util.Log
import io.ktor.client.HttpClient
import io.ktor.client.plugins.websocket.webSocket
import io.ktor.websocket.Frame
import io.ktor.websocket.WebSocketSession
import io.ktor.websocket.readText
import io.picoclaw.android.core.data.remote.dto.WsIncoming
import io.picoclaw.android.core.data.remote.dto.WsOutgoing
import io.picoclaw.android.core.domain.model.ConnectionState
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

class WebSocketClient(
    private val client: HttpClient,
    private val scope: CoroutineScope
) {

    private val _connectionState = MutableStateFlow(ConnectionState.DISCONNECTED)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private val _incomingMessages = MutableSharedFlow<WsOutgoing>(extraBufferCapacity = 64)
    val incomingMessages: SharedFlow<WsOutgoing> = _incomingMessages.asSharedFlow()

    private var session: WebSocketSession? = null
    private var connectJob: Job? = null
    private val json = Json { ignoreUnknownKeys = true }

    var wsUrl: String = "ws://127.0.0.1:18793/ws"

    fun connect() {
        if (connectJob?.isActive == true) return
        connectJob = scope.launch {
            var retryDelay = INITIAL_DELAY
            while (isActive) {
                try {
                    _connectionState.value = ConnectionState.CONNECTING
                    client.webSocket(wsUrl) {
                        session = this
                        _connectionState.value = ConnectionState.CONNECTED
                        retryDelay = INITIAL_DELAY
                        for (frame in incoming) {
                            if (frame is Frame.Text) {
                                val text = frame.readText()
                                try {
                                    val msg = json.decodeFromString<WsOutgoing>(text)
                                    _incomingMessages.emit(msg)
                                } catch (e: Exception) {
                                    Log.w(TAG, "Failed to parse WebSocket message", e)
                                }
                            }
                        }
                    }
                } catch (e: Exception) {
                    Log.w(TAG, "WebSocket connection error", e)
                }
                session = null
                _connectionState.value = ConnectionState.RECONNECTING
                delay(retryDelay)
                retryDelay = (retryDelay * 2).coerceAtMost(MAX_DELAY)
            }
        }
    }

    fun disconnect() {
        connectJob?.cancel()
        connectJob = null
        session = null
        _connectionState.value = ConnectionState.DISCONNECTED
    }

    suspend fun send(dto: WsIncoming): Boolean {
        return try {
            session?.send(Frame.Text(json.encodeToString(dto)))
            true
        } catch (e: Exception) {
            Log.w(TAG, "Failed to send WebSocket message", e)
            false
        }
    }

    companion object {
        private const val TAG = "WebSocketClient"
        private const val INITIAL_DELAY = 1000L
        private const val MAX_DELAY = 30000L
    }
}
