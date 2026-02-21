package io.picoclaw.android.feature.chat.voice

import android.content.Context
import android.graphics.Bitmap
import android.util.Log
import io.picoclaw.android.core.domain.model.ImageAttachment
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import java.io.File

class ScreenCaptureManager(
    private val screenshotSource: ScreenshotSource,
    private val context: Context,
    private val setOverlayVisibility: (Boolean) -> Unit
) {

    val isAvailable: Boolean get() = screenshotSource.isAvailable

    suspend fun captureFrame(): ImageAttachment? {
        if (!screenshotSource.isAvailable) return null
        return try {
            withContext(Dispatchers.Main) {
                setOverlayVisibility(false)
            }
            delay(150)
            val bitmap = screenshotSource.takeScreenshot() ?: return null
            withContext(Dispatchers.IO) {
                try {
                    val imagesDir = File(context.cacheDir, "images").apply { mkdirs() }
                    val file = File(imagesDir, "screen_cap_${System.currentTimeMillis()}.jpg")
                    file.outputStream().use { bitmap.compress(Bitmap.CompressFormat.JPEG, 80, it) }
                    bitmap.recycle()

                    val uri = androidx.core.content.FileProvider.getUriForFile(
                        context,
                        "${context.packageName}.fileprovider",
                        file
                    )
                    ImageAttachment(uri = uri.toString())
                } catch (e: Exception) {
                    bitmap.recycle()
                    throw e
                }
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to capture screen frame", e)
            null
        } finally {
            withContext(Dispatchers.Main) {
                setOverlayVisibility(true)
            }
        }
    }

    companion object {
        private const val TAG = "ScreenCaptureManager"
    }
}
