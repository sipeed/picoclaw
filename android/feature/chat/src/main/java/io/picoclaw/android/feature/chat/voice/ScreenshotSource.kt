package io.picoclaw.android.feature.chat.voice

import android.graphics.Bitmap

interface ScreenshotSource {
    val isAvailable: Boolean
    suspend fun takeScreenshot(): Bitmap?
}
