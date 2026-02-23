package io.clawdroid.core.domain.repository

import io.clawdroid.core.domain.model.TtsConfig
import kotlinx.coroutines.flow.Flow

interface TtsSettingsRepository {
    val ttsConfig: Flow<TtsConfig>
    suspend fun updateEngine(packageName: String?)
    suspend fun updateVoiceName(voiceName: String?)
    suspend fun updateSpeechRate(rate: Float)
    suspend fun updatePitch(pitch: Float)
}
