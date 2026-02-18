package io.picoclaw.android.feature.chat

import io.picoclaw.android.core.domain.model.ImageAttachment

sealed interface ChatEvent {
    data class OnInputChanged(val text: String) : ChatEvent
    data object OnSendClick : ChatEvent
    data class OnImageAdded(val image: ImageAttachment) : ChatEvent
    data class OnImageRemoved(val index: Int) : ChatEvent
    data object OnLoadMore : ChatEvent
    data class OnError(val message: String) : ChatEvent
    data object OnErrorDismissed : ChatEvent
}
