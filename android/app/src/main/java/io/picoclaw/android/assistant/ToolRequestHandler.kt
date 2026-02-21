package io.picoclaw.android.assistant

import android.content.Context
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.util.Log
import io.picoclaw.android.core.data.remote.dto.ToolRequest
import io.picoclaw.android.core.data.remote.dto.ToolResponse
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull

class ToolRequestHandler(
    private val context: Context,
    private val deviceController: DeviceController,
    private val onAccessibilityNeeded: () -> Unit
) {

    suspend fun handle(request: ToolRequest): ToolResponse {
        return try {
            when (request.action) {
                "list_apps" -> handleListApps(request)
                "app_info" -> handleAppInfo(request)
                "launch_app" -> handleLaunchApp(request)
                "current_activity" -> handleCurrentActivity(request)
                "tap" -> handleTap(request)
                "swipe" -> handleSwipe(request)
                "text" -> handleText(request)
                "keyevent" -> handleKeyEvent(request)
                "broadcast" -> handleBroadcast(request)
                "intent" -> handleIntent(request)
                else -> ToolResponse(
                    requestId = request.requestId,
                    success = false,
                    error = "Unknown action: ${request.action}"
                )
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error handling tool request: ${request.action}", e)
            ToolResponse(
                requestId = request.requestId,
                success = false,
                error = "Error: ${e.message}"
            )
        }
    }

    private fun requireAccessibility(request: ToolRequest): ToolResponse? {
        if (!deviceController.isAvailable) {
            onAccessibilityNeeded()
            return ToolResponse(
                requestId = request.requestId,
                success = false,
                error = "accessibility_required"
            )
        }
        return null
    }

    private fun handleListApps(request: ToolRequest): ToolResponse {
        val pm = context.packageManager
        val apps = pm.getInstalledApplications(PackageManager.GET_META_DATA)
            .filter { pm.getLaunchIntentForPackage(it.packageName) != null }
            .map { app ->
                val label = pm.getApplicationLabel(app).toString()
                "${label} (${app.packageName})"
            }
            .sorted()

        return ToolResponse(
            requestId = request.requestId,
            success = true,
            result = "Installed apps (${apps.size}):\n${apps.joinToString("\n")}"
        )
    }

    private fun handleAppInfo(request: ToolRequest): ToolResponse {
        val packageName = request.params?.get("package_name")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "package_name required")

        val pm = context.packageManager
        return try {
            val info = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                pm.getPackageInfo(packageName, PackageManager.PackageInfoFlags.of(0))
            } else {
                @Suppress("DEPRECATION")
                pm.getPackageInfo(packageName, 0)
            }
            val appInfo = info.applicationInfo
            val label = appInfo?.let { pm.getApplicationLabel(it).toString() } ?: packageName
            val isSystem = appInfo?.flags?.and(ApplicationInfo.FLAG_SYSTEM) != 0

            val sb = StringBuilder()
            sb.appendLine("App: $label")
            sb.appendLine("Package: $packageName")
            sb.appendLine("Version: ${info.versionName ?: "unknown"}")
            sb.appendLine("System app: $isSystem")
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                sb.appendLine("Version code: ${info.longVersionCode}")
            }

            ToolResponse(request.requestId, true, result = sb.toString())
        } catch (e: PackageManager.NameNotFoundException) {
            ToolResponse(request.requestId, false, error = "Package not found: $packageName")
        }
    }

    private fun handleLaunchApp(request: ToolRequest): ToolResponse {
        val packageName = request.params?.get("package_name")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "package_name required")

        val intent = context.packageManager.getLaunchIntentForPackage(packageName)
            ?: return ToolResponse(request.requestId, false, error = "No launch intent for $packageName")

        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        context.startActivity(intent)
        return ToolResponse(request.requestId, true, result = "Launched $packageName")
    }

    private suspend fun handleCurrentActivity(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val pkg = deviceController.getCurrentPackage()
            ?: return ToolResponse(request.requestId, false, error = "Could not get current activity")

        return ToolResponse(request.requestId, true, result = "Current package: $pkg")
    }

    private suspend fun handleTap(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val x = request.params?.get("x")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "x coordinate required")
        val y = request.params?.get("y")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "y coordinate required")

        val success = deviceController.tap(x, y)
        return ToolResponse(
            request.requestId, success,
            result = if (success) "Tapped at ($x, $y)" else null,
            error = if (!success) "Tap failed" else null
        )
    }

    private suspend fun handleSwipe(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val x = request.params?.get("x")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "x coordinate required")
        val y = request.params?.get("y")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "y coordinate required")
        val x2 = request.params?.get("x2")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "x2 coordinate required")
        val y2 = request.params?.get("y2")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "y2 coordinate required")
        val durationMs = request.params?.get("duration_ms")?.jsonPrimitive?.longOrNull ?: 300L

        val success = deviceController.swipe(x, y, x2, y2, durationMs)
        return ToolResponse(
            request.requestId, success,
            result = if (success) "Swiped from ($x,$y) to ($x2,$y2)" else null,
            error = if (!success) "Swipe failed" else null
        )
    }

    private suspend fun handleText(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val text = request.params?.get("text")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "text required")

        val success = deviceController.inputText(text)
        return ToolResponse(
            request.requestId, success,
            result = if (success) "Text input: $text" else null,
            error = if (!success) "Text input failed (no focused input field?)" else null
        )
    }

    private fun handleKeyEvent(request: ToolRequest): ToolResponse {
        if (!deviceController.isAvailable) {
            onAccessibilityNeeded()
            return ToolResponse(request.requestId, false, error = "accessibility_required")
        }

        val key = request.params?.get("key")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "key required")

        val success = when (key) {
            "back" -> deviceController.pressBack()
            "home" -> deviceController.pressHome()
            "recents" -> deviceController.pressRecents()
            else -> return ToolResponse(request.requestId, false, error = "Unknown key: $key")
        }
        return ToolResponse(
            request.requestId, success,
            result = if (success) "Key pressed: $key" else null,
            error = if (!success) "Key event failed" else null
        )
    }

    private fun handleBroadcast(request: ToolRequest): ToolResponse {
        val action = request.params?.get("intent_action")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "intent_action required")

        val intent = Intent(action)
        applyExtras(intent, request.params)
        context.sendBroadcast(intent)
        return ToolResponse(request.requestId, true, result = "Broadcast sent: $action")
    }

    private fun handleIntent(request: ToolRequest): ToolResponse {
        val action = request.params?.get("intent_action")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "intent_action required")

        val intent = Intent(action).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)

        request.params?.get("intent_data")?.jsonPrimitive?.contentOrNull?.let {
            intent.data = Uri.parse(it)
        }
        request.params?.get("intent_package")?.jsonPrimitive?.contentOrNull?.let {
            intent.setPackage(it)
        }
        request.params?.get("intent_type")?.jsonPrimitive?.contentOrNull?.let {
            intent.type = it
        }
        applyExtras(intent, request.params)

        return try {
            context.startActivity(intent)
            ToolResponse(request.requestId, true, result = "Intent started: $action")
        } catch (e: Exception) {
            ToolResponse(request.requestId, false, error = "Failed to start intent: ${e.message}")
        }
    }

    private fun applyExtras(intent: Intent, params: JsonObject?) {
        val extras = params?.get("intent_extras") as? JsonObject ?: return
        for ((key, value) in extras) {
            val prim = value as? JsonPrimitive ?: continue
            when {
                prim.isString -> intent.putExtra(key, prim.content)
                prim.content.toBooleanStrictOrNull() != null ->
                    intent.putExtra(key, prim.content.toBooleanStrict())
                prim.longOrNull != null -> intent.putExtra(key, prim.longOrNull!!)
                prim.doubleOrNull != null -> intent.putExtra(key, prim.doubleOrNull!!)
            }
        }
    }

    companion object {
        private const val TAG = "ToolRequestHandler"
    }
}
