package io.clawdroid.core.data.repository

import android.util.Log
import io.clawdroid.core.data.local.ImageFileStorage
import io.clawdroid.core.data.local.dao.MessageDao
import io.clawdroid.core.data.mapper.MessageMapper
import io.clawdroid.core.data.remote.WebSocketClient
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.WsIncoming
import io.clawdroid.core.domain.model.ChatMessage
import io.clawdroid.core.domain.model.ConnectionState
import io.clawdroid.core.domain.model.ImageAttachment
import io.clawdroid.core.domain.model.MessageStatus
import io.clawdroid.core.domain.repository.ChatRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class ChatRepositoryImpl(
    private val webSocketClient: WebSocketClient,
    private val messageDao: MessageDao,
    private val scope: CoroutineScope,
    private val imageFileStorage: ImageFileStorage
) : ChatRepository {

    private val json = Json { ignoreUnknownKeys = true }
    var onToolRequest: (suspend (ToolRequest) -> String)? = null

    private val _displayLimit = MutableStateFlow(INITIAL_LOAD_COUNT)
    private val _statusLabel = MutableStateFlow<String?>(null)

    @Suppress("OPT_IN_USAGE")
    override val messages: StateFlow<List<ChatMessage>> =
        _displayLimit.flatMapLatest { limit ->
            messageDao.getRecentMessages(limit)
        }.map { entities ->
            entities.map { MessageMapper.toDomain(it) }
        }.stateIn(scope, SharingStarted.Lazily, emptyList())

    override val connectionState: StateFlow<ConnectionState> = webSocketClient.connectionState

    override val statusLabel: StateFlow<String?> = _statusLabel.asStateFlow()

    init {
        scope.launch {
            webSocketClient.incomingMessages.collect { dto ->
                when (dto.type) {
                    "status" -> _statusLabel.value = dto.content
                    "status_end" -> _statusLabel.value = null
                    "tool_request" -> handleToolRequest(dto.content)
                    "exit" -> { /* ignored in chat mode */ }
                    else -> {
                        _statusLabel.value = null
                        val entity = MessageMapper.toEntity(dto)
                        messageDao.insert(entity)
                    }
                }
            }
        }
    }

    override suspend fun sendMessage(text: String, images: List<ImageAttachment>, inputMode: String?) {
        val results = images.map { imageFileStorage.saveFromUri(it.uri) }
        val entity = MessageMapper.toEntity(text, results.map { it.imageData }, MessageStatus.SENDING)
        messageDao.insert(entity)
        val wsDto = MessageMapper.toWsIncoming(text, results.map { it.base64 }, inputMode)
        val success = webSocketClient.send(wsDto)
        messageDao.update(entity.copy(status = if (success) MessageStatus.SENT.name else MessageStatus.FAILED.name))
    }

    override fun loadMore() {
        _displayLimit.update { it + PAGE_SIZE }
    }

    override fun connect() {
        webSocketClient.connect()
    }

    override fun disconnect() {
        webSocketClient.disconnect()
    }

    private fun handleToolRequest(content: String) {
        scope.launch {
            try {
                val request = json.decodeFromString<ToolRequest>(content)
                val callback = onToolRequest
                val resultContent = if (callback != null) {
                    callback(request)
                } else {
                    "tool request handler not configured"
                }
                val response = WsIncoming(
                    content = resultContent,
                    type = "tool_response",
                    requestId = request.requestId
                )
                webSocketClient.send(response)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to handle tool request", e)
            }
        }
    }

    companion object {
        private const val TAG = "ChatRepositoryImpl"
        const val INITIAL_LOAD_COUNT = 50
        const val PAGE_SIZE = 30
    }
}
