package io.clawdroid.feature.chat

import io.clawdroid.core.domain.model.TtsConfig
import io.clawdroid.core.domain.model.TtsEngineInfo
import io.clawdroid.core.domain.model.TtsVoiceInfo

data class SettingsUiState(
    val ttsConfig: TtsConfig = TtsConfig(),
    val availableEngines: List<TtsEngineInfo> = emptyList(),
    val availableVoices: List<TtsVoiceInfo> = emptyList(),
    val isTesting: Boolean = false
)
