package io.clawdroid.core.data.repository

import android.content.Context
import android.speech.tts.TextToSpeech
import io.clawdroid.core.domain.model.TtsConfig
import io.clawdroid.core.domain.model.TtsEngineInfo
import io.clawdroid.core.domain.model.TtsVoiceInfo
import io.clawdroid.core.domain.repository.TtsCatalogRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.distinctUntilChangedBy
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

class TtsCatalogRepositoryImpl(
    private val context: Context,
    private val ttsConfigFlow: Flow<TtsConfig>,
    private val scope: CoroutineScope
) : TtsCatalogRepository {

    private val mutex = Mutex()
    private var tts: TextToSpeech? = null
    private var initialized = false
    private var currentEnginePackage: String? = null

    private val _availableEngines = MutableStateFlow<List<TtsEngineInfo>>(emptyList())
    override val availableEngines: StateFlow<List<TtsEngineInfo>> = _availableEngines.asStateFlow()

    private val _availableVoices = MutableStateFlow<List<TtsVoiceInfo>>(emptyList())
    override val availableVoices: StateFlow<List<TtsVoiceInfo>> = _availableVoices.asStateFlow()

    init {
        initTtsEngine(null)

        scope.launch {
            ttsConfigFlow
                .distinctUntilChangedBy { it.enginePackageName }
                .collect { config ->
                    mutex.withLock {
                        if (config.enginePackageName != currentEnginePackage) {
                            initTtsEngine(config.enginePackageName)
                        }
                    }
                }
        }
    }

    private fun initTtsEngine(enginePackageName: String?) {
        tts?.shutdown()
        initialized = false
        currentEnginePackage = enginePackageName

        val listener = TextToSpeech.OnInitListener { status ->
            if (status == TextToSpeech.SUCCESS) {
                initialized = true
                loadAvailableEngines()
                loadAvailableVoices()
            }
        }

        tts = if (enginePackageName != null) {
            TextToSpeech(context, listener, enginePackageName)
        } else {
            TextToSpeech(context, listener)
        }
    }

    private fun loadAvailableEngines() {
        val engine = tts ?: return
        _availableEngines.value = engine.engines.map { info ->
            TtsEngineInfo(
                packageName = info.name,
                label = info.label
            )
        }
    }

    private fun loadAvailableVoices() {
        val engine = tts ?: return
        val voices = engine.voices ?: return
        _availableVoices.value = voices
            .filter { !it.isNetworkConnectionRequired }
            .sortedBy { it.locale.displayName }
            .map { voice ->
                TtsVoiceInfo(
                    name = voice.name,
                    displayLabel = "${voice.locale.displayName} - ${voice.name}",
                    locale = voice.locale.toString()
                )
            }
    }
}
