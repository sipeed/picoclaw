package io.clawdroid.core.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ToolResponse(
    @SerialName("request_id") val requestId: String,
    val success: Boolean,
    val result: String? = null,
    val error: String? = null
)
