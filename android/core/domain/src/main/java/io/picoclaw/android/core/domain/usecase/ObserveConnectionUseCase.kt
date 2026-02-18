package io.picoclaw.android.core.domain.usecase

import io.picoclaw.android.core.domain.model.ConnectionState
import io.picoclaw.android.core.domain.repository.ChatRepository
import kotlinx.coroutines.flow.StateFlow

class ObserveConnectionUseCase(private val repository: ChatRepository) {
    operator fun invoke(): StateFlow<ConnectionState> = repository.connectionState
}
