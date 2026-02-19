package io.picoclaw.android.feature.chat

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.picoclaw.android.core.domain.usecase.ConnectChatUseCase
import io.picoclaw.android.core.domain.usecase.DisconnectChatUseCase
import io.picoclaw.android.core.domain.usecase.LoadMoreMessagesUseCase
import io.picoclaw.android.core.domain.usecase.ObserveConnectionUseCase
import io.picoclaw.android.core.domain.usecase.ObserveMessagesUseCase
import io.picoclaw.android.core.domain.usecase.ObserveStatusUseCase
import io.picoclaw.android.core.domain.usecase.SendMessageUseCase
import io.picoclaw.android.feature.chat.voice.VoiceModeManager
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

class ChatViewModel(
    private val sendMessage: SendMessageUseCase,
    private val observeMessages: ObserveMessagesUseCase,
    private val observeConnection: ObserveConnectionUseCase,
    private val observeStatus: ObserveStatusUseCase,
    private val loadMoreMessages: LoadMoreMessagesUseCase,
    private val connectChat: ConnectChatUseCase,
    private val disconnectChat: DisconnectChatUseCase,
    private val voiceModeManager: VoiceModeManager
) : ViewModel() {

    private val _uiState = MutableStateFlow(ChatUiState())
    val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()

    private val _scrollToBottom = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val scrollToBottom = _scrollToBottom.asSharedFlow()

    init {
        connectChat()

        viewModelScope.launch {
            observeMessages().collect { messages ->
                _uiState.update { it.copy(messages = messages) }
            }
        }

        viewModelScope.launch {
            observeConnection().collect { state ->
                _uiState.update { it.copy(connectionState = state) }
            }
        }

        viewModelScope.launch {
            observeStatus().collect { label ->
                _uiState.update { it.copy(statusLabel = label) }
            }
        }

        viewModelScope.launch {
            voiceModeManager.state.collect { voiceState ->
                _uiState.update { it.copy(voiceModeState = voiceState) }
            }
        }
    }

    fun onEvent(event: ChatEvent) {
        when (event) {
            is ChatEvent.OnInputChanged -> {
                _uiState.update { it.copy(inputText = event.text) }
            }
            is ChatEvent.OnSendClick -> {
                val state = _uiState.value
                val text = state.inputText.trim()
                if (text.isEmpty() && state.pendingImages.isEmpty()) return
                _scrollToBottom.tryEmit(Unit)
                _uiState.update { it.copy(inputText = "", pendingImages = emptyList()) }
                viewModelScope.launch {
                    try {
                        sendMessage(text, state.pendingImages)
                    } catch (e: Exception) {
                        _uiState.update { it.copy(error = e.message) }
                    }
                }
            }
            is ChatEvent.OnImageAdded -> {
                _uiState.update { it.copy(pendingImages = it.pendingImages + event.image) }
            }
            is ChatEvent.OnImageRemoved -> {
                _uiState.update {
                    it.copy(pendingImages = it.pendingImages.filterIndexed { i, _ -> i != event.index })
                }
            }
            is ChatEvent.OnLoadMore -> {
                loadMoreMessages()
            }
            is ChatEvent.OnError -> {
                _uiState.update { it.copy(error = event.message) }
            }
            is ChatEvent.OnErrorDismissed -> {
                _uiState.update { it.copy(error = null) }
            }
            is ChatEvent.OnVoiceModeStart -> {
                voiceModeManager.start(viewModelScope)
            }
            is ChatEvent.OnVoiceModeStop -> {
                voiceModeManager.stop()
            }
            is ChatEvent.OnVoiceModeInterrupt -> {
                voiceModeManager.interrupt()
            }
        }
    }

    override fun onCleared() {
        super.onCleared()
        voiceModeManager.destroy()
        disconnectChat()
    }
}
