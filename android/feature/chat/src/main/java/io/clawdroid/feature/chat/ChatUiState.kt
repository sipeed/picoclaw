package io.clawdroid.feature.chat

import io.clawdroid.core.domain.model.ChatMessage
import io.clawdroid.core.domain.model.ConnectionState
import io.clawdroid.core.domain.model.ImageAttachment
import io.clawdroid.feature.chat.voice.VoiceModeState

data class ChatUiState(
    val messages: List<ChatMessage> = emptyList(),
    val connectionState: ConnectionState = ConnectionState.DISCONNECTED,
    val inputText: String = "",
    val pendingImages: List<ImageAttachment> = emptyList(),
    val isLoadingMore: Boolean = false,
    val canLoadMore: Boolean = true,
    val error: String? = null,
    val statusLabel: String? = null,
    val voiceModeState: VoiceModeState = VoiceModeState()
)
