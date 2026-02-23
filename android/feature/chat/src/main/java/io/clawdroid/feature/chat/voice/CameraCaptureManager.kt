package io.clawdroid.feature.chat.voice

import android.content.Context
import android.graphics.Bitmap
import android.util.Log
import androidx.camera.core.CameraSelector
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.core.content.ContextCompat
import androidx.lifecycle.LifecycleOwner
import io.clawdroid.core.domain.model.ImageAttachment
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

class CameraCaptureManager(private val context: Context) {

    private var cameraProvider: ProcessCameraProvider? = null
    private var currentPreviewView: PreviewView? = null

    fun bind(lifecycleOwner: LifecycleOwner, previewView: PreviewView) {
        if (currentPreviewView === previewView && cameraProvider != null) return
        unbind()
        currentPreviewView = previewView

        val providerFuture = ProcessCameraProvider.getInstance(context)
        providerFuture.addListener({
            try {
                // Stale callback – unbind() was called while waiting
                if (currentPreviewView !== previewView) return@addListener

                val provider = providerFuture.get()
                cameraProvider = provider

                val preview = Preview.Builder().build().also {
                    it.surfaceProvider = previewView.surfaceProvider
                }

                provider.unbindAll()
                provider.bindToLifecycle(
                    lifecycleOwner,
                    CameraSelector.DEFAULT_BACK_CAMERA,
                    preview
                )
            } catch (e: Exception) {
                Log.w(TAG, "Failed to bind camera", e)
            }
        }, ContextCompat.getMainExecutor(context))
    }

    fun unbind() {
        cameraProvider?.unbindAll()
        cameraProvider = null
        currentPreviewView = null
    }

    suspend fun captureFrame(): ImageAttachment? {
        val bitmap = currentPreviewView?.bitmap ?: return null
        return try {
            withContext(Dispatchers.IO) {
                val imagesDir = File(context.cacheDir, "images").apply { mkdirs() }
                val file = File(imagesDir, "voice_cam_${System.currentTimeMillis()}.jpg")
                file.outputStream().use { bitmap.compress(Bitmap.CompressFormat.JPEG, 80, it) }
                val uri = androidx.core.content.FileProvider.getUriForFile(
                    context,
                    "${context.packageName}.fileprovider",
                    file
                )
                ImageAttachment(uri = uri.toString())
            }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to capture frame", e)
            null
        }
    }

    companion object {
        private const val TAG = "CameraCaptureManager"
    }
}
