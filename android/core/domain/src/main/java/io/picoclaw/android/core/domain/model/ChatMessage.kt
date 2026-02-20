package io.picoclaw.android.core.domain.model

data class ChatMessage(
    val id: String,
    val content: String,
    val sender: MessageSender,
    val images: List<ImageData> = emptyList(),
    val timestamp: Long,
    val status: MessageStatus,
    val messageType: String? = null
)

enum class MessageSender { USER, AGENT }

enum class MessageStatus { SENDING, SENT, FAILED, RECEIVED }
