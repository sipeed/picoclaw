package io.picoclaw.android.assistant

import android.accessibilityservice.AccessibilityService
import android.accessibilityservice.GestureDescription
import android.graphics.Path
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

class DeviceController {

    @Volatile
    private var service: AccessibilityService? = null

    val isAvailable: Boolean get() = service != null

    fun setService(s: AccessibilityService) {
        service = s
    }

    fun clearService() {
        service = null
    }

    suspend fun tap(x: Float, y: Float): Boolean {
        val svc = service ?: return false
        val path = Path().apply { moveTo(x, y) }
        val stroke = GestureDescription.StrokeDescription(path, 0, 100)
        val gesture = GestureDescription.Builder().addStroke(stroke).build()
        return dispatchGesture(svc, gesture)
    }

    suspend fun swipe(x1: Float, y1: Float, x2: Float, y2: Float, durationMs: Long = 300): Boolean {
        val svc = service ?: return false
        val path = Path().apply {
            moveTo(x1, y1)
            lineTo(x2, y2)
        }
        val stroke = GestureDescription.StrokeDescription(path, 0, durationMs)
        val gesture = GestureDescription.Builder().addStroke(stroke).build()
        return dispatchGesture(svc, gesture)
    }

    fun pressBack(): Boolean {
        return service?.performGlobalAction(AccessibilityService.GLOBAL_ACTION_BACK) ?: false
    }

    fun pressHome(): Boolean {
        return service?.performGlobalAction(AccessibilityService.GLOBAL_ACTION_HOME) ?: false
    }

    fun pressRecents(): Boolean {
        return service?.performGlobalAction(AccessibilityService.GLOBAL_ACTION_RECENTS) ?: false
    }

    fun getCurrentPackage(): String? {
        val svc = service ?: return null
        val ownPackage = svc.packageName
        // Skip our own overlay window and find the actual foreground app
        return svc.windows
            .filter { it.type == android.view.accessibility.AccessibilityWindowInfo.TYPE_APPLICATION }
            .firstOrNull { it.root?.packageName?.toString() != ownPackage }
            ?.root?.packageName?.toString()
    }

    suspend fun inputText(text: String): Boolean {
        val svc = service ?: return false
        val ownPackage = svc.packageName
        // Search non-overlay windows for the focused input field
        val focusedNode = svc.windows
            .filter { it.type == android.view.accessibility.AccessibilityWindowInfo.TYPE_APPLICATION }
            .firstNotNullOfOrNull { win ->
                val root = win.root ?: return@firstNotNullOfOrNull null
                if (root.packageName?.toString() == ownPackage) return@firstNotNullOfOrNull null
                root.findFocus(AccessibilityNodeInfo.FOCUS_INPUT)
            } ?: return false
        val args = Bundle().apply {
            putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE, text)
        }
        return focusedNode.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, args)
    }

    private suspend fun dispatchGesture(
        svc: AccessibilityService,
        gesture: GestureDescription
    ): Boolean = suspendCancellableCoroutine { cont ->
        svc.dispatchGesture(
            gesture,
            object : AccessibilityService.GestureResultCallback() {
                override fun onCompleted(gestureDescription: GestureDescription?) {
                    cont.resume(true)
                }

                override fun onCancelled(gestureDescription: GestureDescription?) {
                    cont.resume(false)
                }
            },
            null
        )
    }
}
