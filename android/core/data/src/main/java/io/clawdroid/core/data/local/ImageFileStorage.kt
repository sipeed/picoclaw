package io.clawdroid.core.data.local

import android.content.Context
import android.graphics.BitmapFactory
import android.net.Uri
import android.util.Base64
import io.clawdroid.core.domain.model.ImageData
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.util.UUID

class ImageFileStorage(private val context: Context) {

    private val imageDir = File(context.filesDir, "chat_images").also { it.mkdirs() }

    data class SaveResult(
        val imageData: ImageData,
        val base64: String
    )

    suspend fun saveFromUri(uriString: String): SaveResult = withContext(Dispatchers.IO) {
        val bytes = context.contentResolver.openInputStream(Uri.parse(uriString))?.use {
            it.readBytes()
        } ?: error("Cannot read URI: $uriString")

        val file = File(imageDir, "${UUID.randomUUID()}.jpg")
        file.writeBytes(bytes)

        val opts = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(file.absolutePath, opts)

        SaveResult(
            imageData = ImageData(file.absolutePath, opts.outWidth, opts.outHeight),
            base64 = Base64.encodeToString(bytes, Base64.NO_WRAP)
        )
    }
}
