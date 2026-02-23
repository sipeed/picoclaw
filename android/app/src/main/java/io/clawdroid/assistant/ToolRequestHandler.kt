package io.clawdroid.assistant

import android.content.Context
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.Rect
import android.net.Uri
import android.os.Build
import android.util.Base64
import android.util.Log
import android.view.accessibility.AccessibilityNodeInfo
import io.clawdroid.core.data.remote.dto.ToolRequest
import io.clawdroid.core.data.remote.dto.ToolResponse
import io.clawdroid.feature.chat.voice.ScreenshotSource
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.longOrNull
import java.io.ByteArrayOutputStream

class ToolRequestHandler(
    private val context: Context,
    private val deviceController: DeviceController,
    private val screenshotSource: ScreenshotSource,
    private val setOverlayVisibility: (Boolean) -> Unit,
    private val onAccessibilityNeeded: () -> Unit
) {

    suspend fun handle(request: ToolRequest): ToolResponse {
        return try {
            when (request.action) {
                "search_apps" -> handleSearchApps(request)
                "app_info" -> handleAppInfo(request)
                "launch_app" -> handleLaunchApp(request)
                "screenshot" -> handleScreenshot(request)
                "get_ui_tree" -> handleGetUiTree(request)
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

    private suspend fun <T> withOverlayHidden(block: suspend () -> T): T {
        return try {
            withContext(Dispatchers.Main) { setOverlayVisibility(false) }
            delay(150)
            block()
        } finally {
            withContext(Dispatchers.Main) { setOverlayVisibility(true) }
        }
    }

    private fun handleSearchApps(request: ToolRequest): ToolResponse {
        val query = request.params?.get("query")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "query required")

        val pm = context.packageManager
        val q = query.lowercase()
        val matches = pm.getInstalledApplications(PackageManager.GET_META_DATA)
            .filter { app ->
                val label = pm.getApplicationLabel(app).toString().lowercase()
                label.contains(q) || app.packageName.lowercase().contains(q)
            }
            .map { app ->
                val label = pm.getApplicationLabel(app).toString()
                val launchable = pm.getLaunchIntentForPackage(app.packageName) != null
                val isSystem = app.flags and ApplicationInfo.FLAG_SYSTEM != 0
                buildString {
                    append("$label (${app.packageName})")
                    if (launchable) append(" [launchable]")
                    if (isSystem) append(" [system]")
                }
            }
            .sorted()

        return if (matches.isEmpty()) {
            ToolResponse(request.requestId, true, result = "No apps found matching \"$query\"")
        } else {
            ToolResponse(
                requestId = request.requestId,
                success = true,
                result = "Found ${matches.size} app(s) matching \"$query\":\n${matches.joinToString("\n")}"
            )
        }
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

    private suspend fun handleScreenshot(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        return withOverlayHidden {
            val bitmap = screenshotSource.takeScreenshot()
                ?: return@withOverlayHidden ToolResponse(request.requestId, false, error = "Screenshot capture failed")
            try {
                val base64 = withContext(Dispatchers.IO) {
                    val stream = ByteArrayOutputStream()
                    bitmap.compress(Bitmap.CompressFormat.JPEG, 80, stream)
                    Base64.encodeToString(stream.toByteArray(), Base64.NO_WRAP)
                }
                ToolResponse(request.requestId, true, result = base64)
            } finally {
                bitmap.recycle()
            }
        }
    }

    private suspend fun handleGetUiTree(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val resourceId = request.params?.get("resource_id")?.jsonPrimitive?.contentOrNull
        val index = request.params?.get("index")?.jsonPrimitive?.intOrNull ?: 0
        val boundsX = request.params?.get("bounds_x")?.jsonPrimitive?.doubleOrNull
        val boundsY = request.params?.get("bounds_y")?.jsonPrimitive?.doubleOrNull
        val maxDepth = request.params?.get("max_depth")?.jsonPrimitive?.intOrNull ?: 15
        val maxNodes = request.params?.get("max_nodes")?.jsonPrimitive?.intOrNull ?: 300

        return withOverlayHidden {
            val root = deviceController.getRootNode()
                ?: return@withOverlayHidden ToolResponse(request.requestId, false, error = "Could not get UI tree")
            try {
                val startNode = resolveStartNode(root, resourceId, index, boundsX, boundsY)
                    ?: return@withOverlayHidden ToolResponse(request.requestId, false, error = buildString {
                        if (resourceId != null) append("No node found with resource_id=$resourceId (index=$index)")
                        else append("No node found at bounds ($boundsX, $boundsY)")
                    })
                try {
                    val sb = StringBuilder()
                    val nodeCount = intArrayOf(0)
                    dumpNode(startNode, sb, 0, maxDepth, maxNodes, nodeCount)
                    if (nodeCount[0] >= maxNodes) {
                        sb.appendLine("[truncated: max_nodes=$maxNodes reached]")
                    }
                    ToolResponse(request.requestId, true, result = sb.toString())
                } finally {
                    if (startNode !== root) startNode.recycle()
                }
            } finally {
                root.recycle()
            }
        }
    }

    private fun resolveStartNode(
        root: AccessibilityNodeInfo,
        resourceId: String?,
        index: Int,
        boundsX: Double?,
        boundsY: Double?
    ): AccessibilityNodeInfo? {
        if (resourceId != null) {
            val matches = root.findAccessibilityNodeInfosByViewId(resourceId)
            if (matches.isNullOrEmpty()) return null
            val target = matches.getOrNull(index)
            // Recycle unused matches
            for ((i, node) in matches.withIndex()) {
                if (i != index) node.recycle()
            }
            return target
        }
        if (boundsX != null && boundsY != null) {
            return findNodeAtPoint(root, boundsX.toInt(), boundsY.toInt())
        }
        return root
    }

    private fun findNodeAtPoint(node: AccessibilityNodeInfo, x: Int, y: Int): AccessibilityNodeInfo? {
        val bounds = Rect()
        node.getBoundsInScreen(bounds)
        if (!bounds.contains(x, y)) return null
        // Find the deepest (smallest) child that contains the point
        for (i in 0 until node.childCount) {
            val child = node.getChild(i) ?: continue
            val found = findNodeAtPoint(child, x, y)
            if (found != null) return found
            child.recycle()
        }
        return node
    }

    private fun dumpNode(
        node: AccessibilityNodeInfo,
        sb: StringBuilder,
        depth: Int,
        maxDepth: Int,
        maxNodes: Int,
        nodeCount: IntArray
    ) {
        if (nodeCount[0] >= maxNodes) return
        // Skip invisible nodes
        if (!node.isVisibleToUser) return
        nodeCount[0]++
        val indent = "  ".repeat(depth)
        val bounds = Rect()
        node.getBoundsInScreen(bounds)

        // Strip common class name prefixes
        val className = node.className?.toString() ?: "View"
        val shortClass = className
            .removePrefix("android.widget.")
            .removePrefix("android.view.")

        sb.append("${indent}[${shortClass}]")

        // Only output non-empty fields
        node.text?.takeIf { it.isNotEmpty() }?.let { sb.append(" text=$it") }
        node.contentDescription?.takeIf { it.isNotEmpty() }?.let { sb.append(" desc=$it") }
        sb.append(" bounds=$bounds")
        // Only output non-default values: clickable=true (default is false), enabled=false (default is true)
        if (node.isClickable) sb.append(" clickable")
        if (!node.isEnabled) sb.append(" enabled=false")
        node.viewIdResourceName?.let { sb.append(" id=$it") }

        sb.appendLine()

        if (depth >= maxDepth) {
            if (node.childCount > 0) {
                sb.appendLine("${indent}  [truncated: ${node.childCount} children at depth $depth]")
            }
            return
        }
        for (i in 0 until node.childCount) {
            if (nodeCount[0] >= maxNodes) return
            val child = node.getChild(i) ?: continue
            try {
                dumpNode(child, sb, depth + 1, maxDepth, maxNodes, nodeCount)
            } finally {
                child.recycle()
            }
        }
    }

    private suspend fun handleTap(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val x = request.params?.get("x")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "x coordinate required")
        val y = request.params?.get("y")?.jsonPrimitive?.doubleOrNull?.toFloat()
            ?: return ToolResponse(request.requestId, false, error = "y coordinate required")

        return withOverlayHidden {
            val success = deviceController.tap(x, y)
            ToolResponse(
                request.requestId, success,
                result = if (success) "Tapped at ($x, $y)" else null,
                error = if (!success) "Tap failed" else null
            )
        }
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

        return withOverlayHidden {
            val success = deviceController.swipe(x, y, x2, y2, durationMs)
            ToolResponse(
                request.requestId, success,
                result = if (success) "Swiped from ($x,$y) to ($x2,$y2)" else null,
                error = if (!success) "Swipe failed" else null
            )
        }
    }

    private suspend fun handleText(request: ToolRequest): ToolResponse {
        requireAccessibility(request)?.let { return it }

        val text = request.params?.get("text")?.jsonPrimitive?.contentOrNull
            ?: return ToolResponse(request.requestId, false, error = "text required")

        return withOverlayHidden {
            val success = deviceController.inputText(text)
            ToolResponse(
                request.requestId, success,
                result = if (success) "Text input: $text" else null,
                error = if (!success) "Text input failed (no focused input field?)" else null
            )
        }
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
