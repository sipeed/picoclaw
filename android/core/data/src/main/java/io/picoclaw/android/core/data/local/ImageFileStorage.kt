package io.picoclaw.android.core.data.local

import android.content.Context
import android.util.Base64
import java.io.File
import java.util.UUID

class ImageFileStorage(context: Context) {

    private val imageDir = File(context.filesDir, "chat_images").also { it.mkdirs() }

    fun saveBase64ToFile(base64: String): String {
        val bytes = Base64.decode(base64, Base64.DEFAULT)
        val file = File(imageDir, "${UUID.randomUUID()}.jpg")
        file.writeBytes(bytes)
        return file.absolutePath
    }
}
