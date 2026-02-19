package io.picoclaw.android.feature.chat.voice

import android.speech.SpeechRecognizer
import io.picoclaw.android.core.domain.model.MessageSender
import io.picoclaw.android.core.domain.model.MessageStatus
import io.picoclaw.android.core.domain.model.VoicePhase
import io.picoclaw.android.core.domain.usecase.ObserveMessagesUseCase
import io.picoclaw.android.core.domain.usecase.ObserveStatusUseCase
import io.picoclaw.android.core.domain.usecase.SendMessageUseCase
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.Job
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.selects.onTimeout
import kotlinx.coroutines.selects.select

@OptIn(ExperimentalCoroutinesApi::class)
class VoiceModeManager(
    private val sttWrapper: SpeechRecognizerWrapper,
    private val ttsWrapper: TextToSpeechWrapper,
    private val sendMessage: SendMessageUseCase,
    private val observeMessages: ObserveMessagesUseCase,
    private val observeStatus: ObserveStatusUseCase
) {

    private val _state = MutableStateFlow(VoiceModeState())
    val state: StateFlow<VoiceModeState> = _state.asStateFlow()

    private var loopJob: Job? = null
    private var parentScope: CoroutineScope? = null

    fun start(scope: CoroutineScope) {
        if (loopJob?.isActive == true) return
        parentScope = scope
        _state.update {
            VoiceModeState(isActive = true, phase = VoicePhase.LISTENING)
        }
        loopJob = scope.launch {
            voiceLoop()
        }
    }

    fun stop() {
        loopJob?.cancel()
        loopJob = null
        parentScope = null
        ttsWrapper.stop()
        _state.value = VoiceModeState()
    }

    fun interrupt() {
        val scope = parentScope ?: return
        if (loopJob?.isActive != true) return

        ttsWrapper.stop()
        loopJob?.cancel()
        loopJob = null

        _state.update {
            VoiceModeState(isActive = true, phase = VoicePhase.LISTENING)
        }
        loopJob = scope.launch { voiceLoop() }
    }

    fun destroy() {
        stop()
    }

    // Fix #5: coroutineScope {} で自身のスコープを生成（CoroutineScope. 拡張関数を廃止）
    private suspend fun voiceLoop() = coroutineScope {
        val spokenIds = mutableSetOf<String>()
        val speechQueue = Channel<String>(Channel.UNLIMITED)

        // Fix #3: knownMessageIds のスナップショットを collectorJob 内に移動し、
        // スナップショットと collect 開始の間のレースウィンドウを排除
        val collectorJob = launch {
            val initialIds = observeMessages().value.map { it.id }.toSet()
            observeMessages().collect { messages ->
                messages.forEach { msg ->
                    if (msg.sender == MessageSender.AGENT &&
                        msg.id !in initialIds &&
                        msg.id !in spokenIds &&
                        msg.status == MessageStatus.RECEIVED
                    ) {
                        spokenIds.add(msg.id)
                        speechQueue.send(msg.content)
                    }
                }
            }
        }

        try {
            while (isActive) {
                _state.update {
                    it.copy(
                        phase = VoicePhase.LISTENING,
                        recognizedText = "", responseText = "",
                        statusText = null,
                        errorMessage = null, amplitudeNormalized = 0f
                    )
                }

                val userTextChannel = Channel<String?>(1)
                val listenJob = launch {
                    val text = listen()
                    userTextChannel.send(text)
                }

                select<Unit> {
                    userTextChannel.onReceive { text ->
                        if (!text.isNullOrBlank()) {
                            _state.update { it.copy(phase = VoicePhase.SENDING, recognizedText = text) }
                            try {
                                sendMessage(text)
                            } catch (e: Exception) {
                                _state.update {
                                    it.copy(phase = VoicePhase.ERROR, errorMessage = "送信に失敗しました")
                                }
                                delay(2000)
                                return@onReceive
                            }
                            awaitAndSpeakResponse(speechQueue)
                        } else {
                            drainSpeechQueue(speechQueue)
                        }
                    }
                    speechQueue.onReceive { content ->
                        listenJob.cancel()
                        speakAndDrain(content, speechQueue)
                    }
                }
            }
        } finally {
            collectorJob.cancel()
            speechQueue.close()
        }
    }

    private sealed interface WaitResult {
        data class Message(val content: String) : WaitResult
        data object Heartbeat : WaitResult
        data object Timeout : WaitResult
    }

    // Fix #1 + #2 + #4 + #5: select で speechQueue / heartbeat / onTimeout(30s) を待つ
    private suspend fun awaitAndSpeakResponse(speechQueue: Channel<String>) = coroutineScope {
        val heartbeat = Channel<Unit>(Channel.CONFLATED)

        val statusJob = launch {
            observeStatus().collect { label ->
                _state.update { it.copy(statusText = label) }      // Fix #4: null も伝播
                if (label != null) heartbeat.trySend(Unit)
            }
        }

        try {
            _state.update { it.copy(phase = VoicePhase.THINKING, statusText = null) }

            while (true) {
                val result = select<WaitResult> {
                    speechQueue.onReceive { WaitResult.Message(it) }
                    heartbeat.onReceive { WaitResult.Heartbeat }
                    onTimeout(30_000) { WaitResult.Timeout }
                }

                when (result) {
                    WaitResult.Timeout -> {
                        _state.update {
                            it.copy(phase = VoicePhase.ERROR, errorMessage = "応答がタイムアウトしました")
                        }
                        delay(2000)
                        return@coroutineScope
                    }
                    WaitResult.Heartbeat -> continue
                    is WaitResult.Message -> {
                        speakAndDrain(result.content, speechQueue)
                        val currentStatus = observeStatus().value
                        if (currentStatus == null) return@coroutineScope
                        _state.update {
                            it.copy(phase = VoicePhase.THINKING, statusText = currentStatus)
                        }
                    }
                }
            }
        } finally {
            statusJob.cancel()
        }
    }

    private suspend fun speakAndDrain(firstContent: String, speechQueue: Channel<String>) {
        _state.update { it.copy(phase = VoicePhase.SPEAKING, responseText = firstContent) }
        ttsWrapper.speak(firstContent)
        while (true) {
            val next = speechQueue.tryReceive().getOrNull() ?: break
            _state.update { it.copy(responseText = next) }
            ttsWrapper.speak(next)
        }
    }

    private suspend fun drainSpeechQueue(speechQueue: Channel<String>) {
        val first = speechQueue.tryReceive().getOrNull() ?: return
        speakAndDrain(first, speechQueue)
    }

    private suspend fun listen(): String? {
        var finalText: String? = null
        sttWrapper.startListening().collect { result ->
            when (result) {
                is SttResult.Partial -> {
                    _state.update { it.copy(recognizedText = result.text) }
                }
                is SttResult.Final -> {
                    finalText = result.text
                }
                is SttResult.RmsChanged -> {
                    val normalized = ((result.rms + 2f) / 12f).coerceIn(0f, 1f)
                    _state.update { it.copy(amplitudeNormalized = normalized) }
                }
                is SttResult.Error -> {
                    if (result.code == SpeechRecognizer.ERROR_NO_MATCH ||
                        result.code == SpeechRecognizer.ERROR_SPEECH_TIMEOUT
                    ) {
                        finalText = ""
                    } else {
                        _state.update {
                            it.copy(
                                phase = VoicePhase.ERROR,
                                errorMessage = "音声認識エラー (code=${result.code})"
                            )
                        }
                        delay(2000)
                        finalText = null
                    }
                }
            }
        }
        return finalText
    }
}
