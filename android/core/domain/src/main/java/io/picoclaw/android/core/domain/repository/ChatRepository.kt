package io.picoclaw.android.core.domain.repository

import io.picoclaw.android.core.domain.model.ChatMessage
import io.picoclaw.android.core.domain.model.ConnectionState
import io.picoclaw.android.core.domain.model.ImageAttachment
import kotlinx.coroutines.flow.StateFlow

interface ChatRepository {
    val messages: StateFlow<List<ChatMessage>>
    val connectionState: StateFlow<ConnectionState>
    suspend fun sendMessage(text: String, images: List<ImageAttachment> = emptyList())
    fun loadMore()
    fun connect()
    fun disconnect()
}
