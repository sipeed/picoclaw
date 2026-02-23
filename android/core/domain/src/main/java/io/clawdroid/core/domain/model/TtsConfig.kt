package io.clawdroid.core.domain.model

data class TtsConfig(
    val enginePackageName: String? = null,
    val voiceName: String? = null,
    val speechRate: Float = 1.0f,
    val pitch: Float = 1.0f
)
