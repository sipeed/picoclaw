package io.clawdroid.core.domain.usecase

import io.clawdroid.core.domain.model.ImageAttachment
import io.clawdroid.core.domain.repository.ChatRepository

class SendMessageUseCase(private val repository: ChatRepository) {
    suspend operator fun invoke(text: String, images: List<ImageAttachment> = emptyList(), inputMode: String? = null) {
        repository.sendMessage(text, images, inputMode)
    }
}
