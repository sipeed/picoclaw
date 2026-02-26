package io.clawdroid.backend.api

data class GatewaySettings(
    val wsPort: Int = 18793,
    val httpPort: Int = 18790,
    val apiKey: String = "",
)
