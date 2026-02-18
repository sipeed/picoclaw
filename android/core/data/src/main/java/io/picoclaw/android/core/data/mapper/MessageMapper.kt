package io.picoclaw.android.core.data.mapper

import io.picoclaw.android.core.data.local.entity.MessageEntity
import io.picoclaw.android.core.data.remote.dto.WsIncoming
import io.picoclaw.android.core.data.remote.dto.WsOutgoing
import io.picoclaw.android.core.domain.model.ChatMessage
import io.picoclaw.android.core.domain.model.ImageAttachment
import io.picoclaw.android.core.domain.model.MessageSender
import io.picoclaw.android.core.domain.model.MessageStatus
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.util.UUID

object MessageMapper {

    fun toDomain(entity: MessageEntity): ChatMessage {
        val images = entity.imageBase64List?.let {
            try {
                Json.decodeFromString<List<String>>(it)
            } catch (_: Exception) {
                emptyList()
            }
        } ?: emptyList()

        return ChatMessage(
            id = entity.id,
            content = entity.content,
            sender = MessageSender.valueOf(entity.sender),
            images = images,
            timestamp = entity.timestamp,
            status = MessageStatus.valueOf(entity.status)
        )
    }

    fun toEntity(dto: WsOutgoing): MessageEntity {
        return MessageEntity(
            id = UUID.randomUUID().toString(),
            content = dto.content,
            sender = MessageSender.AGENT.name,
            imageBase64List = null,
            timestamp = System.currentTimeMillis(),
            status = MessageStatus.RECEIVED.name
        )
    }

    fun toEntity(text: String, images: List<ImageAttachment>, status: MessageStatus): MessageEntity {
        val imageJson = if (images.isNotEmpty()) {
            Json.encodeToString(images.map { it.base64 })
        } else null

        return MessageEntity(
            id = UUID.randomUUID().toString(),
            content = text,
            sender = MessageSender.USER.name,
            imageBase64List = imageJson,
            timestamp = System.currentTimeMillis(),
            status = status.name
        )
    }

    fun toWsIncoming(text: String, images: List<ImageAttachment>): WsIncoming {
        return WsIncoming(
            content = text,
            images = if (images.isNotEmpty()) images.map { it.base64 } else null
        )
    }
}
