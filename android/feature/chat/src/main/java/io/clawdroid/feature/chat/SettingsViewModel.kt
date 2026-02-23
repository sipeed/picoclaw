package io.clawdroid.feature.chat

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.clawdroid.core.domain.repository.TtsCatalogRepository
import io.clawdroid.core.domain.repository.TtsSettingsRepository
import io.clawdroid.feature.chat.voice.TextToSpeechWrapper
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class SettingsViewModel(
    private val ttsSettingsRepository: TtsSettingsRepository,
    private val ttsCatalogRepository: TtsCatalogRepository,
    private val ttsWrapper: TextToSpeechWrapper
) : ViewModel() {

    private val _uiState = MutableStateFlow(SettingsUiState())
    val uiState: StateFlow<SettingsUiState> = _uiState.asStateFlow()

    init {
        viewModelScope.launch {
            ttsSettingsRepository.ttsConfig.collect { config ->
                _uiState.update { it.copy(ttsConfig = config) }
            }
        }
        viewModelScope.launch {
            ttsCatalogRepository.availableEngines.collect { engines ->
                _uiState.update { it.copy(availableEngines = engines) }
            }
        }
        viewModelScope.launch {
            ttsCatalogRepository.availableVoices.collect { voices ->
                _uiState.update { it.copy(availableVoices = voices) }
            }
        }
    }

    fun onEngineSelected(packageName: String?) {
        viewModelScope.launch { ttsSettingsRepository.updateEngine(packageName) }
    }

    fun onVoiceSelected(voiceName: String?) {
        viewModelScope.launch { ttsSettingsRepository.updateVoiceName(voiceName) }
    }

    fun onSpeechRateChanged(rate: Float) {
        viewModelScope.launch { ttsSettingsRepository.updateSpeechRate(rate) }
    }

    fun onPitchChanged(pitch: Float) {
        viewModelScope.launch { ttsSettingsRepository.updatePitch(pitch) }
    }

    fun onTestSpeak() {
        viewModelScope.launch {
            _uiState.update { it.copy(isTesting = true) }
            ttsWrapper.speak("これはテスト音声です。This is a test.")
            _uiState.update { it.copy(isTesting = false) }
        }
    }
}
