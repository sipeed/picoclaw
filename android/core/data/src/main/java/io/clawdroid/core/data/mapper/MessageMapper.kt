package io.clawdroid.core.data.mapper

import android.util.Log
import io.clawdroid.core.data.local.entity.MessageEntity
import io.clawdroid.core.data.remote.dto.WsIncoming
import io.clawdroid.core.data.remote.dto.WsOutgoing
import io.clawdroid.core.domain.model.ChatMessage
import io.clawdroid.core.domain.model.ImageData
import io.clawdroid.core.domain.model.MessageSender
import io.clawdroid.core.domain.model.MessageStatus
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.util.UUID

@Serializable
private data class ImageEntry(val path: String, val width: Int, val height: Int)

object MessageMapper {

    fun toDomain(entity: MessageEntity): ChatMessage {
        val images = entity.imagePathList?.let {
            try {
                Json.decodeFromString<List<ImageEntry>>(it).map { e ->
                    ImageData(e.path, e.width, e.height)
                }
            } catch (e: Exception) {
                Log.w("MessageMapper", "Failed to parse image path list", e)
                emptyList()
            }
        } ?: emptyList()

        return ChatMessage(
            id = entity.id,
            content = entity.content,
            sender = MessageSender.valueOf(entity.sender),
            images = images,
            timestamp = entity.timestamp,
            status = MessageStatus.valueOf(entity.status),
            messageType = entity.messageType
        )
    }

    fun toEntity(dto: WsOutgoing): MessageEntity {
        return MessageEntity(
            id = UUID.randomUUID().toString(),
            content = dto.content,
            sender = MessageSender.AGENT.name,
            imagePathList = null,
            timestamp = System.currentTimeMillis(),
            status = MessageStatus.RECEIVED.name,
            messageType = dto.type
        )
    }

    fun toEntity(text: String, images: List<ImageData>, status: MessageStatus): MessageEntity {
        val pathJson = if (images.isNotEmpty()) {
            Json.encodeToString(images.map { ImageEntry(it.path, it.width, it.height) })
        } else null

        return MessageEntity(
            id = UUID.randomUUID().toString(),
            content = text,
            sender = MessageSender.USER.name,
            imagePathList = pathJson,
            timestamp = System.currentTimeMillis(),
            status = status.name
        )
    }

    fun toWsIncoming(text: String, base64Images: List<String>, inputMode: String? = null): WsIncoming {
        return WsIncoming(
            content = text,
            images = base64Images.ifEmpty { null },
            inputMode = inputMode
        )
    }
}
