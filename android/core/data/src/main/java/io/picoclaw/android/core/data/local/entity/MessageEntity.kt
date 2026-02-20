package io.picoclaw.android.core.data.local.entity

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "messages")
data class MessageEntity(
    @PrimaryKey val id: String,
    val content: String,
    val sender: String,
    val imagePathList: String?,
    val timestamp: Long,
    val status: String,
    val messageType: String? = null
)
