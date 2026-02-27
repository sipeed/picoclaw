package io.clawdroid.backend.api

data class GatewaySettings(
    val httpPort: Int = 18790,
    val apiKey: String = "",
) {
    val httpBaseUrl: String get() = "http://127.0.0.1:$httpPort"
}
