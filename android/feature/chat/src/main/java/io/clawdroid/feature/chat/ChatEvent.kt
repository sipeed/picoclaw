package io.clawdroid.feature.chat

import io.clawdroid.core.domain.model.ImageAttachment

sealed interface ChatEvent {
    data class OnInputChanged(val text: String) : ChatEvent
    data object OnSendClick : ChatEvent
    data class OnImageAdded(val image: ImageAttachment) : ChatEvent
    data class OnImageRemoved(val index: Int) : ChatEvent
    data object OnLoadMore : ChatEvent
    data class OnError(val message: String) : ChatEvent
    data object OnErrorDismissed : ChatEvent
    data object OnVoiceModeStart : ChatEvent
    data object OnVoiceModeStop : ChatEvent
    data object OnVoiceModeInterrupt : ChatEvent
    data object OnVoiceCameraToggle : ChatEvent
}
