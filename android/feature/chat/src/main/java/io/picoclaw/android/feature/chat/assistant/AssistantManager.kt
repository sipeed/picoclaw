package io.picoclaw.android.feature.chat.assistant

import android.content.ContentResolver
import android.speech.SpeechRecognizer
import android.util.Base64
import android.util.Log
import io.picoclaw.android.core.domain.model.AssistantMessage
import io.picoclaw.android.core.domain.model.VoicePhase
import io.picoclaw.android.core.domain.repository.AssistantConnection
import io.picoclaw.android.feature.chat.voice.CameraCaptureManager
import io.picoclaw.android.feature.chat.voice.SpeechRecognizerWrapper
import io.picoclaw.android.feature.chat.voice.SttResult
import io.picoclaw.android.feature.chat.voice.TextToSpeechWrapper
import io.picoclaw.android.feature.chat.voice.ChatTurn
import io.picoclaw.android.feature.chat.voice.VoiceModeState
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
class AssistantManager(
    private val sttWrapper: SpeechRecognizerWrapper,
    private val ttsWrapper: TextToSpeechWrapper,
    private val connection: AssistantConnection,
    private val cameraCaptureManager: CameraCaptureManager,
    private val contentResolver: ContentResolver
) {

    private val _state = MutableStateFlow(VoiceModeState())
    val state: StateFlow<VoiceModeState> = _state.asStateFlow()

    private var loopJob: Job? = null
    private var parentScope: CoroutineScope? = null

    fun toggleCamera() {
        _state.update { it.copy(isCameraActive = !it.isCameraActive) }
    }

    fun start(scope: CoroutineScope) {
        if (loopJob?.isActive == true) return
        parentScope = scope
        _state.update {
            VoiceModeState(isActive = true, phase = VoicePhase.LISTENING, isCameraActive = it.isCameraActive, chatHistory = it.chatHistory)
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
        cameraCaptureManager.unbind()
        _state.value = VoiceModeState()
    }

    fun interrupt() {
        val scope = parentScope ?: return
        if (loopJob?.isActive != true) return

        ttsWrapper.stop()
        loopJob?.cancel()
        loopJob = null

        _state.update {
            VoiceModeState(isActive = true, phase = VoicePhase.LISTENING, isCameraActive = it.isCameraActive, chatHistory = it.chatHistory)
        }
        loopJob = scope.launch { voiceLoop() }
    }

    fun destroy() {
        stop()
    }

    private suspend fun voiceLoop() = coroutineScope {
        val speechQueue = Channel<String>(Channel.UNLIMITED)

        val messageCollectorJob = launch {
            connection.messages.collect { msg: AssistantMessage ->
                if (msg.type == "warning" || msg.type == "error") {
                    _state.update { it.copy(responseText = msg.content) }
                } else {
                    speechQueue.send(msg.content)
                }
            }
        }

        val statusCollectorJob = launch {
            connection.statusText.collect { label ->
                _state.update { it.copy(statusText = label) }
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
                            _state.update { it.copy(phase = VoicePhase.SENDING, recognizedText = text, chatHistory = it.chatHistory + ChatTurn("user", text)) }
                            try {
                                val base64Images = if (_state.value.isCameraActive) {
                                    captureAndEncode()
                                } else emptyList()
                                connection.send(text, base64Images)
                            } catch (e: Exception) {
                                Log.w(TAG, "Failed to send message", e)
                                _state.update {
                                    it.copy(phase = VoicePhase.ERROR, errorMessage = "Failed to send")
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
            messageCollectorJob.cancel()
            statusCollectorJob.cancel()
            speechQueue.close()
        }
    }

    private sealed interface WaitResult {
        data class Message(val content: String) : WaitResult
        data object Heartbeat : WaitResult
        data object Timeout : WaitResult
    }

    private suspend fun awaitAndSpeakResponse(
        speechQueue: Channel<String>
    ) = coroutineScope {
        val heartbeat = Channel<Unit>(Channel.CONFLATED)

        val statusJob = launch {
            connection.statusText.collect { label ->
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
                            it.copy(phase = VoicePhase.ERROR, errorMessage = "Response timed out")
                        }
                        delay(2000)
                        return@coroutineScope
                    }
                    WaitResult.Heartbeat -> continue
                    is WaitResult.Message -> {
                        speakAndDrain(result.content, speechQueue)
                        val currentStatus = connection.statusText.value
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
        val chunks = mutableListOf(firstContent)
        try {
            _state.update { it.copy(phase = VoicePhase.SPEAKING, responseText = firstContent) }
            ttsWrapper.speak(firstContent)
            while (true) {
                val next = speechQueue.tryReceive().getOrNull() ?: break
                chunks.add(next)
                _state.update { it.copy(responseText = next) }
                ttsWrapper.speak(next)
            }
        } finally {
            val fullResponse = chunks.joinToString("\n")
            _state.update { it.copy(chatHistory = it.chatHistory + ChatTurn("assistant", fullResponse)) }
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
                                errorMessage = "Speech recognition error (code=${result.code})"
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

    private suspend fun captureAndEncode(): List<String> {
        val attachment = cameraCaptureManager.captureFrame() ?: return emptyList()
        val uri = android.net.Uri.parse(attachment.uri)
        return try {
            val bytes = contentResolver.openInputStream(uri)?.use { it.readBytes() }
                ?: return emptyList()
            listOf(Base64.encodeToString(bytes, Base64.NO_WRAP))
        } catch (e: Exception) {
            Log.w(TAG, "Failed to encode captured image", e)
            emptyList()
        }
    }

    companion object {
        private const val TAG = "AssistantManager"
    }
}
