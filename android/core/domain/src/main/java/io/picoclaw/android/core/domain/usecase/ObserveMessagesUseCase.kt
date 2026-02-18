package io.picoclaw.android.core.domain.usecase

import io.picoclaw.android.core.domain.model.ChatMessage
import io.picoclaw.android.core.domain.repository.ChatRepository
import kotlinx.coroutines.flow.StateFlow

class ObserveMessagesUseCase(private val repository: ChatRepository) {
    operator fun invoke(): StateFlow<List<ChatMessage>> = repository.messages
}
