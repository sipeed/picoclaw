package io.clawdroid.core.domain.usecase

import io.clawdroid.core.domain.repository.ChatRepository

class LoadMoreMessagesUseCase(private val repository: ChatRepository) {
    operator fun invoke() {
        repository.loadMore()
    }
}
