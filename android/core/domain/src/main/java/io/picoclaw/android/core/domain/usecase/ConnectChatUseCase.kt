package io.picoclaw.android.core.domain.usecase

import io.picoclaw.android.core.domain.repository.ChatRepository

class ConnectChatUseCase(private val repository: ChatRepository) {
    operator fun invoke() {
        repository.connect()
    }
}
