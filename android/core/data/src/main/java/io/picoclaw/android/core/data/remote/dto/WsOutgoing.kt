package io.picoclaw.android.core.data.remote.dto

import kotlinx.serialization.Serializable

@Serializable
data class WsOutgoing(
    val content: String,
    val type: String? = null
)
