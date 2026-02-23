package io.clawdroid.feature.chat.voice

import io.clawdroid.core.domain.model.VoicePhase

data class ChatTurn(val role: String, val text: String)

data class VoiceModeState(
    val isActive: Boolean = false,
    val phase: VoicePhase = VoicePhase.IDLE,
    val recognizedText: String = "",
    val responseText: String = "",
    val statusText: String? = null,
    val errorMessage: String? = null,
    val amplitudeNormalized: Float = 0f,
    val isCameraActive: Boolean = false,
    val isScreenCaptureActive: Boolean = false,
    val chatHistory: List<ChatTurn> = emptyList()
)
