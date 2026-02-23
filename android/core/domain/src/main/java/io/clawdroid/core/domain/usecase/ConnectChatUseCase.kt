package io.clawdroid.core.domain.usecase

import io.clawdroid.core.domain.repository.ChatRepository

class ConnectChatUseCase(private val repository: ChatRepository) {
    operator fun invoke() {
        repository.connect()
    }
}
