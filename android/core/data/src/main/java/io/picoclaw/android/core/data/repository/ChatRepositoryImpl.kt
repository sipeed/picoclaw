package io.picoclaw.android.core.data.repository

import io.picoclaw.android.core.data.local.ImageFileStorage
import io.picoclaw.android.core.data.local.dao.MessageDao
import io.picoclaw.android.core.data.mapper.MessageMapper
import io.picoclaw.android.core.data.remote.WebSocketClient
import io.picoclaw.android.core.domain.model.ChatMessage
import io.picoclaw.android.core.domain.model.ConnectionState
import io.picoclaw.android.core.domain.model.ImageAttachment
import io.picoclaw.android.core.domain.model.MessageStatus
import io.picoclaw.android.core.domain.repository.ChatRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class ChatRepositoryImpl(
    private val webSocketClient: WebSocketClient,
    private val messageDao: MessageDao,
    private val scope: CoroutineScope,
    private val imageFileStorage: ImageFileStorage
) : ChatRepository {

    private val _displayLimit = MutableStateFlow(INITIAL_LOAD_COUNT)

    @Suppress("OPT_IN_USAGE")
    override val messages: StateFlow<List<ChatMessage>> =
        _displayLimit.flatMapLatest { limit ->
            messageDao.getRecentMessages(limit)
        }.map { entities ->
            entities.map { MessageMapper.toDomain(it) }
        }.stateIn(scope, SharingStarted.Lazily, emptyList())

    override val connectionState: StateFlow<ConnectionState> = webSocketClient.connectionState

    init {
        scope.launch {
            webSocketClient.incomingMessages.collect { dto ->
                val entity = MessageMapper.toEntity(dto)
                messageDao.insert(entity)
            }
        }
    }

    override suspend fun sendMessage(text: String, images: List<ImageAttachment>) {
        val imagePaths = images.map { imageFileStorage.saveBase64ToFile(it.base64) }
        val entity = MessageMapper.toEntity(text, imagePaths, MessageStatus.SENDING)
        messageDao.insert(entity)
        val wsDto = MessageMapper.toWsIncoming(text, images)
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

    companion object {
        const val INITIAL_LOAD_COUNT = 50
        const val PAGE_SIZE = 30
    }
}
