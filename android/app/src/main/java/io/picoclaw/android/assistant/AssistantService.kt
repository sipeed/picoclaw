package io.picoclaw.android.assistant

import android.Manifest
import android.app.Notification
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.graphics.PixelFormat
import android.os.IBinder
import android.provider.Settings
import android.view.Gravity
import android.view.MotionEvent
import android.view.View
import android.view.WindowManager
import android.widget.FrameLayout
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInWindow
import androidx.compose.ui.platform.ComposeView
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import androidx.lifecycle.LifecycleService
import androidx.lifecycle.setViewTreeLifecycleOwner
import androidx.savedstate.SavedStateRegistry
import androidx.savedstate.SavedStateRegistryController
import androidx.savedstate.SavedStateRegistryOwner
import androidx.savedstate.setViewTreeSavedStateRegistryOwner
import io.ktor.client.HttpClient
import io.picoclaw.android.PermissionRequestActivity
import io.picoclaw.android.core.data.remote.WebSocketClient
import io.picoclaw.android.core.data.repository.AssistantConnectionImpl
import io.picoclaw.android.core.domain.repository.AssistantConnection
import io.picoclaw.android.core.domain.repository.TtsSettingsRepository
import io.picoclaw.android.core.ui.theme.PicoClawTheme
import io.picoclaw.android.feature.chat.assistant.AssistantManager
import io.picoclaw.android.feature.chat.assistant.AssistantPillBar
import io.picoclaw.android.feature.chat.voice.CameraCaptureManager
import io.picoclaw.android.feature.chat.voice.ScreenCaptureManager
import io.picoclaw.android.feature.chat.voice.ScreenshotSource
import io.picoclaw.android.feature.chat.voice.SpeechRecognizerWrapper
import io.picoclaw.android.feature.chat.voice.TextToSpeechWrapper
import io.picoclaw.android.receiver.NotificationHelper
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import org.koin.android.ext.android.inject

class AssistantService : LifecycleService(), SavedStateRegistryOwner {

    private val httpClient: HttpClient by inject()
    private val ttsSettingsRepo: TtsSettingsRepository by inject()
    private val screenshotSource: ScreenshotSource by inject()
    private val deviceController: DeviceController by inject()

    private lateinit var serviceScope: CoroutineScope
    private lateinit var connection: AssistantConnection
    private lateinit var assistantManager: AssistantManager
    private lateinit var toolRequestHandler: ToolRequestHandler
    private lateinit var ttsWrapper: TextToSpeechWrapper
    private lateinit var sttWrapper: SpeechRecognizerWrapper
    private lateinit var cameraCaptureManager: CameraCaptureManager
    private lateinit var screenCaptureManager: ScreenCaptureManager

    private var showAccessibilityGuide by mutableStateOf(false)
    private var overlayAtTop by mutableStateOf(false)
    private var overlayView: View? = null
    private val windowManager by lazy { getSystemService(WINDOW_SERVICE) as WindowManager }

    private val savedStateRegistryController = SavedStateRegistryController.create(this)
    override val savedStateRegistry: SavedStateRegistry
        get() = savedStateRegistryController.savedStateRegistry

    private val permissionReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context, intent: Intent) {
            val permission = intent.getStringExtra(PermissionRequestActivity.EXTRA_PERMISSION)
            val granted = intent.getBooleanExtra(PermissionRequestActivity.EXTRA_GRANTED, false)
            if (permission == Manifest.permission.CAMERA && granted) {
                startForeground(NOTIFICATION_ID, buildNotification(), computeForegroundTypes())
                assistantManager.toggleCamera()
            }
        }
    }

    override fun onCreate() {
        savedStateRegistryController.performAttach()
        savedStateRegistryController.performRestore(null)
        super.onCreate()

        serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

        connection = AssistantConnectionImpl(httpClient)

        toolRequestHandler = ToolRequestHandler(
            context = applicationContext,
            deviceController = deviceController,
            screenshotSource = screenshotSource,
            setOverlayVisibility = { visible -> setOverlayVisible(visible) },
            onAccessibilityNeeded = { showAccessibilityGuide = true }
        )
        (connection as AssistantConnectionImpl).onToolRequest = { request ->
            val response = toolRequestHandler.handle(request)
            if (response.success) {
                response.result ?: ""
            } else {
                response.error ?: "unknown error"
            }
        }
        (connection as AssistantConnectionImpl).onExit = { farewell ->
            handleExitCommand(farewell)
        }

        sttWrapper = SpeechRecognizerWrapper(this)
        ttsWrapper = TextToSpeechWrapper(this, ttsSettingsRepo.ttsConfig)
        cameraCaptureManager = CameraCaptureManager(this)
        screenCaptureManager = ScreenCaptureManager(screenshotSource, applicationContext) { visible ->
            setOverlayVisible(visible)
        }

        assistantManager = AssistantManager(
            sttWrapper = sttWrapper,
            ttsWrapper = ttsWrapper,
            connection = connection,
            cameraCaptureManager = cameraCaptureManager,
            screenCaptureManager = screenCaptureManager,
            contentResolver = contentResolver
        )

        ContextCompat.registerReceiver(
            this,
            permissionReceiver,
            IntentFilter(PermissionRequestActivity.ACTION_RESULT),
            ContextCompat.RECEIVER_NOT_EXPORTED
        )
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)

        startForeground(
            NOTIFICATION_ID,
            buildNotification(),
            computeForegroundTypes()
        )

        // Resolve wsUrl from the main WebSocketClient
        val mainWsClient: WebSocketClient by inject()
        connection.connect(mainWsClient.wsUrl)

        addOverlay()
        assistantManager.start(serviceScope)

        return START_NOT_STICKY
    }

    override fun onDestroy() {
        unregisterReceiver(permissionReceiver)
        removeOverlay()
        assistantManager.destroy()
        ttsWrapper.destroy()
        connection.disconnect()
        serviceScope.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent): IBinder? {
        super.onBind(intent)
        return null
    }

    private fun handleCameraToggle() {
        if (assistantManager.state.value.isCameraActive) {
            assistantManager.toggleCamera()
            return
        }
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
            == PackageManager.PERMISSION_GRANTED
        ) {
            startForeground(NOTIFICATION_ID, buildNotification(), computeForegroundTypes())
            assistantManager.toggleCamera()
        } else {
            startActivity(PermissionRequestActivity.intent(this, Manifest.permission.CAMERA))
        }
    }

    private fun handleScreenCaptureToggle() {
        if (assistantManager.state.value.isScreenCaptureActive) {
            assistantManager.toggleScreenCapture()
            return
        }
        if (screenCaptureManager.isAvailable) {
            // Turn off camera first if active
            if (assistantManager.state.value.isCameraActive) {
                assistantManager.toggleCamera()
            }
            assistantManager.toggleScreenCapture()
        } else {
            showAccessibilityGuide = true
        }
    }

    private fun shutdown() {
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun handleExitCommand(farewell: String?) {
        if (!farewell.isNullOrBlank()) {
            serviceScope.launch {
                ttsWrapper.speak(farewell)
                shutdown()
            }
        } else {
            shutdown()
        }
    }

    private fun moveOverlayTo(top: Boolean) {
        val view = overlayView ?: return
        val lp = view.layoutParams as? WindowManager.LayoutParams ?: return
        lp.gravity = if (top) Gravity.TOP else Gravity.BOTTOM
        windowManager.updateViewLayout(view, lp)
        overlayAtTop = top
    }

    private fun setOverlayVisible(visible: Boolean) {
        val view = overlayView ?: return
        val lp = view.layoutParams as? WindowManager.LayoutParams ?: return
        if (visible) {
            lp.flags = lp.flags and WindowManager.LayoutParams.FLAG_NOT_TOUCHABLE.inv()
            view.visibility = View.VISIBLE
        } else {
            lp.flags = lp.flags or WindowManager.LayoutParams.FLAG_NOT_TOUCHABLE
            view.visibility = View.INVISIBLE
        }
        windowManager.updateViewLayout(view, lp)
    }

    private fun addOverlay() {
        if (overlayView != null) return

        val density = resources.displayMetrics.density
        val fixedHeightPx = (350 * density).toInt()

        val params = WindowManager.LayoutParams(
            WindowManager.LayoutParams.MATCH_PARENT,
            fixedHeightPx,
            WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY,
            WindowManager.LayoutParams.FLAG_NOT_TOUCH_MODAL or
                WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE or
                WindowManager.LayoutParams.FLAG_LAYOUT_IN_SCREEN,
            PixelFormat.TRANSLUCENT
        ).apply {
            gravity = if (overlayAtTop) Gravity.TOP else Gravity.BOTTOM
        }

        val wrapper = object : FrameLayout(this@AssistantService) {
            @Volatile var contentTop = fixedHeightPx
            @Volatile var contentBottom = Int.MAX_VALUE
            private var gestureInContent = false
            override fun dispatchTouchEvent(ev: MotionEvent): Boolean {
                if (ev.actionMasked == MotionEvent.ACTION_DOWN) {
                    gestureInContent = ev.y >= contentTop && ev.y <= contentBottom
                }
                if (!gestureInContent) return false
                return super.dispatchTouchEvent(ev)
            }
        }

        wrapper.setViewTreeLifecycleOwner(this)
        wrapper.setViewTreeSavedStateRegistryOwner(this)

        ComposeView(this).apply {
            setContent {
                PicoClawTheme {
                    val state by assistantManager.state.collectAsState()
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = if (overlayAtTop) Alignment.TopCenter else Alignment.BottomCenter
                    ) {
                        AssistantPillBar(
                            state = state,
                            isAtTop = overlayAtTop,
                            onClose = { shutdown() },
                            onInterrupt = { assistantManager.interrupt() },
                            onPositionChange = { top -> moveOverlayTo(top) },
                            onCameraToggle = { handleCameraToggle() },
                            onScreenCaptureToggle = { handleScreenCaptureToggle() },
                            cameraCaptureManager = cameraCaptureManager,
                            modifier = Modifier.onGloballyPositioned { coordinates ->
                                wrapper.contentTop = coordinates.positionInWindow().y.toInt()
                                wrapper.contentBottom = (coordinates.positionInWindow().y + coordinates.size.height).toInt()
                            }
                        )
                    }

                    if (showAccessibilityGuide) {
                        Box(
                            modifier = Modifier
                                .fillMaxSize()
                                .background(MaterialTheme.colorScheme.scrim.copy(alpha = 0.5f))
                                .clickable(
                                    interactionSource = remember { MutableInteractionSource() },
                                    indication = null
                                ) { showAccessibilityGuide = false },
                            contentAlignment = Alignment.Center
                        ) {
                            Surface(
                                shape = MaterialTheme.shapes.extraLarge,
                                tonalElevation = 6.dp,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 24.dp)
                                    .clickable(
                                        interactionSource = remember { MutableInteractionSource() },
                                        indication = null
                                    ) {}
                            ) {
                                Column(modifier = Modifier.padding(24.dp)) {
                                    Text(
                                        text = "画面キャプチャの設定",
                                        style = MaterialTheme.typography.headlineSmall
                                    )
                                    Text(
                                        text = "画面キャプチャを使用するには、ユーザー補助の設定で" +
                                            " PicoClaw を有効にしてください。\n\n" +
                                            "有効にできない場合は、アプリ情報から" +
                                            "「制限付き設定を許可」を実行してから" +
                                            "再度お試しください。",
                                        style = MaterialTheme.typography.bodyMedium,
                                        modifier = Modifier.padding(top = 16.dp)
                                    )
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(top = 24.dp),
                                        horizontalArrangement = Arrangement.End
                                    ) {
                                        TextButton(onClick = { showAccessibilityGuide = false }) {
                                            Text("キャンセル")
                                        }
                                        TextButton(onClick = {
                                            showAccessibilityGuide = false
                                            val intent = Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS)
                                                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                                            startActivity(intent)
                                        }) {
                                            Text("設定を開く")
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            wrapper.addView(this, FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            ))
        }

        windowManager.addView(wrapper, params)
        overlayView = wrapper
    }

    private fun removeOverlay() {
        overlayView?.let {
            windowManager.removeView(it)
            overlayView = null
        }
    }

    private fun computeForegroundTypes(): Int {
        var types = ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE or
            ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
            == PackageManager.PERMISSION_GRANTED
        ) {
            types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
        }
        return types
    }

    private fun buildNotification(): Notification {
        return NotificationCompat.Builder(this, NotificationHelper.ASSISTANT_CHANNEL_ID)
            .setSmallIcon(android.R.drawable.ic_btn_speak_now)
            .setContentTitle("PicoClaw Assistant")
            .setContentText("Listening...")
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setOngoing(true)
            .build()
    }

    companion object {
        private const val NOTIFICATION_ID = 2001
    }
}
