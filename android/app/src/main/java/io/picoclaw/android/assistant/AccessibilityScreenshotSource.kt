package io.picoclaw.android.assistant

import android.accessibilityservice.AccessibilityService
import android.graphics.Bitmap
import android.util.Log
import android.view.Display
import io.picoclaw.android.feature.chat.voice.ScreenshotSource
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

class AccessibilityScreenshotSource : ScreenshotSource {

    @Volatile
    private var service: AccessibilityService? = null

    override val isAvailable: Boolean get() = service != null

    fun setService(s: AccessibilityService) {
        service = s
    }

    fun clearService() {
        service = null
    }

    override suspend fun takeScreenshot(): Bitmap? {
        val svc = service ?: return null
        return suspendCancellableCoroutine { cont ->
            svc.takeScreenshot(
                Display.DEFAULT_DISPLAY,
                svc.mainExecutor,
                object : AccessibilityService.TakeScreenshotCallback {
                    override fun onSuccess(result: AccessibilityService.ScreenshotResult) {
                        val hwBitmap = Bitmap.wrapHardwareBuffer(
                            result.hardwareBuffer, result.colorSpace
                        )
                        result.hardwareBuffer.close()
                        val swBitmap = hwBitmap?.copy(Bitmap.Config.ARGB_8888, false)
                        hwBitmap?.recycle()
                        cont.resume(swBitmap)
                    }

                    override fun onFailure(errorCode: Int) {
                        Log.w(TAG, "takeScreenshot failed with errorCode=$errorCode")
                        cont.resume(null)
                    }
                }
            )
        }
    }

    companion object {
        private const val TAG = "A11yScreenshotSource"
    }
}
