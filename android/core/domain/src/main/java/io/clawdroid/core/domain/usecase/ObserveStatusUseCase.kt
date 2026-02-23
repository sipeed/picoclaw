package io.clawdroid.core.domain.usecase

import io.clawdroid.core.domain.repository.ChatRepository
import kotlinx.coroutines.flow.StateFlow

class ObserveStatusUseCase(private val repository: ChatRepository) {
    operator fun invoke(): StateFlow<String?> = repository.statusLabel
}
