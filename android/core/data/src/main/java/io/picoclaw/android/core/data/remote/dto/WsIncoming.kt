package io.picoclaw.android.core.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class WsIncoming(
    val content: String,
    @SerialName("sender_id") val senderId: String? = null,
    val images: List<String>? = null,
    @SerialName("input_mode") val inputMode: String? = null,
    val type: String? = null,
    @SerialName("request_id") val requestId: String? = null
)
