package io.picoclaw.android.assistant

import android.accessibilityservice.AccessibilityService
import android.view.accessibility.AccessibilityEvent
import org.koin.android.ext.android.inject

class PicoClawAccessibilityService : AccessibilityService() {

    private val screenshotSource: AccessibilityScreenshotSource by inject()
    private val deviceController: DeviceController by inject()

    override fun onServiceConnected() {
        super.onServiceConnected()
        screenshotSource.setService(this)
        deviceController.setService(this)
    }

    override fun onDestroy() {
        screenshotSource.clearService()
        deviceController.clearService()
        super.onDestroy()
    }

    override fun onAccessibilityEvent(event: AccessibilityEvent?) {
        // No-op
    }

    override fun onInterrupt() {
        // No-op
    }
}
