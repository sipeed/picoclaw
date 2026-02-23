package io.clawdroid.feature.chat.voice

import android.content.Context
import android.speech.tts.TextToSpeech
import android.speech.tts.UtteranceProgressListener
import io.clawdroid.core.domain.model.TtsConfig
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.distinctUntilChangedBy
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.util.UUID
import kotlin.coroutines.resume

class TextToSpeechWrapper(
    private val context: Context,
    ttsConfigFlow: Flow<TtsConfig>
) {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private val mutex = Mutex()

    private var tts: TextToSpeech? = null
    private var initialized = false
    private var currentConfig = TtsConfig()
    private var currentEnginePackage: String? = null

    init {
        initTtsEngine(null)

        scope.launch {
            ttsConfigFlow.collect { config ->
                mutex.withLock { currentConfig = config }
            }
        }

        scope.launch {
            ttsConfigFlow
                .distinctUntilChangedBy { it.enginePackageName }
                .collect { config ->
                    mutex.withLock {
                        if (config.enginePackageName != currentEnginePackage) {
                            switchEngine(config.enginePackageName)
                        }
                    }
                }
        }
    }

    private fun initTtsEngine(enginePackageName: String?) {
        tts?.stop()
        tts?.shutdown()
        initialized = false
        currentEnginePackage = enginePackageName

        val listener = TextToSpeech.OnInitListener { status ->
            if (status == TextToSpeech.SUCCESS) {
                initialized = true
            }
        }

        tts = if (enginePackageName != null) {
            TextToSpeech(context, listener, enginePackageName)
        } else {
            TextToSpeech(context, listener)
        }
    }

    private fun switchEngine(enginePackageName: String?) {
        initTtsEngine(enginePackageName)
    }

    private fun applyConfig(engine: TextToSpeech) {
        engine.setSpeechRate(currentConfig.speechRate)
        engine.setPitch(currentConfig.pitch)
        currentConfig.voiceName?.let { name ->
            engine.voices?.firstOrNull { it.name == name }?.let { engine.voice = it }
        }
    }

    suspend fun speak(text: String): Boolean = suspendCancellableCoroutine { cont ->
        val engine = tts
        if (engine == null || !initialized) {
            cont.resume(false)
            return@suspendCancellableCoroutine
        }

        applyConfig(engine)

        val utteranceId = UUID.randomUUID().toString()

        engine.setOnUtteranceProgressListener(object : UtteranceProgressListener() {
            override fun onStart(id: String?) {}

            override fun onDone(id: String?) {
                if (id == utteranceId && cont.isActive) {
                    cont.resume(true)
                }
            }

            @Deprecated("Deprecated in Java")
            override fun onError(id: String?) {
                if (id == utteranceId && cont.isActive) {
                    cont.resume(false)
                }
            }

            override fun onError(id: String?, errorCode: Int) {
                if (id == utteranceId && cont.isActive) {
                    cont.resume(false)
                }
            }
        })

        cont.invokeOnCancellation {
            engine.stop()
        }

        engine.speak(text, TextToSpeech.QUEUE_FLUSH, null, utteranceId)
    }

    fun stop() {
        tts?.stop()
    }

    fun destroy() {
        scope.cancel()
        tts?.stop()
        tts?.shutdown()
        tts = null
    }
}
