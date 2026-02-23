package io.clawdroid.core.domain.usecase

import io.clawdroid.core.domain.model.ChatMessage
import io.clawdroid.core.domain.repository.ChatRepository
import kotlinx.coroutines.flow.StateFlow

class ObserveMessagesUseCase(private val repository: ChatRepository) {
    operator fun invoke(): StateFlow<List<ChatMessage>> = repository.messages
}
