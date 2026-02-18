package io.picoclaw.android.core.domain.model

data class ChatMessage(
    val id: String,
    val content: String,
    val sender: MessageSender,
    val images: List<String> = emptyList(),
    val timestamp: Long,
    val status: MessageStatus
)

enum class MessageSender { USER, AGENT }

enum class MessageStatus { SENDING, SENT, FAILED, RECEIVED }
