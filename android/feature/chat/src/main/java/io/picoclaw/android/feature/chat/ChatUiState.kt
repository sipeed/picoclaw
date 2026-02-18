package io.picoclaw.android.feature.chat

import io.picoclaw.android.core.domain.model.ChatMessage
import io.picoclaw.android.core.domain.model.ConnectionState
import io.picoclaw.android.core.domain.model.ImageAttachment

data class ChatUiState(
    val messages: List<ChatMessage> = emptyList(),
    val connectionState: ConnectionState = ConnectionState.DISCONNECTED,
    val inputText: String = "",
    val pendingImages: List<ImageAttachment> = emptyList(),
    val isLoadingMore: Boolean = false,
    val canLoadMore: Boolean = true,
    val error: String? = null
)
