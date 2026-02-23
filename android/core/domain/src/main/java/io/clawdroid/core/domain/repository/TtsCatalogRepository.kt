package io.clawdroid.core.domain.repository

import io.clawdroid.core.domain.model.TtsEngineInfo
import io.clawdroid.core.domain.model.TtsVoiceInfo
import kotlinx.coroutines.flow.StateFlow

interface TtsCatalogRepository {
    val availableEngines: StateFlow<List<TtsEngineInfo>>
    val availableVoices: StateFlow<List<TtsVoiceInfo>>
}
