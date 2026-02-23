package io.clawdroid.core.data.repository

import android.util.Log
import io.ktor.client.HttpClient
import io.clawdroid.core.data.remote.WebSocketClient
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.WsIncoming
import io.clawdroid.core.domain.model.AssistantMessage
import io.clawdroid.core.domain.model.ConnectionState
import io.clawdroid.core.domain.repository.AssistantConnection
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import java.util.UUID

typealias ToolRequestCallback = suspend (ToolRequest) -> String

class AssistantConnectionImpl(
    private val httpClient: HttpClient
) : AssistantConnection {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val clientId = UUID.randomUUID().toString()
    private val wsClient = WebSocketClient(httpClient, scope, clientId, "assistant")
    private val json = Json { ignoreUnknownKeys = true }

    private val _messages = MutableSharedFlow<AssistantMessage>(extraBufferCapacity = 64)
    override val messages: SharedFlow<AssistantMessage> = _messages.asSharedFlow()

    private val _statusText = MutableStateFlow<String?>(null)
    override val statusText: StateFlow<String?> = _statusText.asStateFlow()

    override val connectionState: StateFlow<ConnectionState> = wsClient.connectionState

    var onToolRequest: ToolRequestCallback? = null
    var onExit: ((String?) -> Unit)? = null

    init {
        scope.launch {
            wsClient.incomingMessages.collect { dto ->
                when (dto.type) {
                    "status" -> _statusText.value = dto.content
                    "status_end" -> _statusText.value = null
                    "tool_request" -> handleToolRequest(dto.content)
                    "exit" -> onExit?.invoke(dto.content)
                    else -> {
                        _messages.emit(AssistantMessage(content = dto.content, type = dto.type))
                    }
                }
            }
        }
    }

    private fun handleToolRequest(content: String) {
        scope.launch {
            try {
                val request = json.decodeFromString<ToolRequest>(content)
                val callback = onToolRequest
                val resultContent = if (callback != null) {
                    callback(request)
                } else {
                    "error: tool request handler not configured"
                }

                val response = WsIncoming(
                    content = resultContent,
                    type = "tool_response",
                    requestId = request.requestId
                )
                wsClient.send(response)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to handle tool request", e)
            }
        }
    }

    override fun connect(wsUrl: String) {
        wsClient.wsUrl = wsUrl
        wsClient.connect()
    }

    override fun disconnect() {
        wsClient.disconnect()
        scope.cancel()
    }

    override suspend fun send(text: String, images: List<String>, inputMode: String) {
        val dto = WsIncoming(
            content = text,
            images = images.ifEmpty { null },
            inputMode = inputMode
        )
        wsClient.send(dto)
    }

    companion object {
        private const val TAG = "AssistantConnectionImpl"
    }
}
