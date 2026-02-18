package io.picoclaw.android.core.domain.usecase

import io.picoclaw.android.core.domain.model.ImageAttachment
import io.picoclaw.android.core.domain.repository.ChatRepository

class SendMessageUseCase(private val repository: ChatRepository) {
    suspend operator fun invoke(text: String, images: List<ImageAttachment> = emptyList()) {
        repository.sendMessage(text, images)
    }
}
