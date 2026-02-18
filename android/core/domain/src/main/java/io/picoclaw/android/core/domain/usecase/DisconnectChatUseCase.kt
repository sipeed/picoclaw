package io.picoclaw.android.core.domain.usecase

import io.picoclaw.android.core.domain.repository.ChatRepository

class DisconnectChatUseCase(private val repository: ChatRepository) {
    operator fun invoke() {
        repository.disconnect()
    }
}
