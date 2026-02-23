package io.clawdroid.core.data.remote.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

@Serializable
data class ToolRequest(
    @SerialName("request_id") val requestId: String,
    val action: String,
    val params: JsonObject? = null
)
