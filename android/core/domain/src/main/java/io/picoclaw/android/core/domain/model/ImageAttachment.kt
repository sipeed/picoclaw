package io.picoclaw.android.core.domain.model

data class ImageAttachment(
    val uri: String? = null,
    val base64: String,
    val mimeType: String = "image/png"
)
