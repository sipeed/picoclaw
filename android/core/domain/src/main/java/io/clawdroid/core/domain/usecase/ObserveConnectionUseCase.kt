package io.clawdroid.core.domain.usecase

import io.clawdroid.core.domain.model.ConnectionState
import io.clawdroid.core.domain.repository.ChatRepository
import kotlinx.coroutines.flow.StateFlow

class ObserveConnectionUseCase(private val repository: ChatRepository) {
    operator fun invoke(): StateFlow<ConnectionState> = repository.connectionState
}
