package io.clawdroid.core.domain.repository

import io.clawdroid.core.domain.model.ChatMessage
import io.clawdroid.core.domain.model.ConnectionState
import io.clawdroid.core.domain.model.ImageAttachment
import kotlinx.coroutines.flow.StateFlow

interface ChatRepository {
    val messages: StateFlow<List<ChatMessage>>
    val connectionState: StateFlow<ConnectionState>
    val statusLabel: StateFlow<String?>
    suspend fun sendMessage(text: String, images: List<ImageAttachment> = emptyList(), inputMode: String? = null)
    fun loadMore()
    fun connect()
    fun disconnect()
}
