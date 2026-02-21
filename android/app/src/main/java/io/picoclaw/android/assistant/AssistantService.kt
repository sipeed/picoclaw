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
import android.view.Gravity
import android.view.MotionEvent
import android.view.View
import android.view.WindowManager
import android.widget.FrameLayout
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
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
import io.picoclaw.android.feature.chat.voice.SpeechRecognizerWrapper
import io.picoclaw.android.feature.chat.voice.TextToSpeechWrapper
import io.picoclaw.android.receiver.NotificationHelper
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import org.koin.android.ext.android.inject

class AssistantService : LifecycleService(), SavedStateRegistryOwner {

    private val httpClient: HttpClient by inject()
    private val ttsSettingsRepo: TtsSettingsRepository by inject()

    private lateinit var serviceScope: CoroutineScope
    private lateinit var connection: AssistantConnection
    private lateinit var assistantManager: AssistantManager
    private lateinit var ttsWrapper: TextToSpeechWrapper
    private lateinit var sttWrapper: SpeechRecognizerWrapper
    private lateinit var cameraCaptureManager: CameraCaptureManager

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

        sttWrapper = SpeechRecognizerWrapper(this)
        ttsWrapper = TextToSpeechWrapper(this, ttsSettingsRepo.ttsConfig)
        cameraCaptureManager = CameraCaptureManager(this)

        assistantManager = AssistantManager(
            sttWrapper = sttWrapper,
            ttsWrapper = ttsWrapper,
            connection = connection,
            cameraCaptureManager = cameraCaptureManager,
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
            ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
                or ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE
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
            assistantManager.toggleCamera()
        } else {
            startActivity(PermissionRequestActivity.intent(this, Manifest.permission.CAMERA))
        }
    }

    private fun shutdown() {
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
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
                WindowManager.LayoutParams.FLAG_LAYOUT_IN_SCREEN,
            PixelFormat.TRANSLUCENT
        ).apply {
            gravity = Gravity.BOTTOM
        }

        val wrapper = object : FrameLayout(this@AssistantService) {
            @Volatile var contentTop = fixedHeightPx
            private var gestureInContent = false
            override fun dispatchTouchEvent(ev: MotionEvent): Boolean {
                if (ev.actionMasked == MotionEvent.ACTION_DOWN) {
                    gestureInContent = ev.y >= contentTop
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
                        contentAlignment = Alignment.BottomCenter
                    ) {
                        AssistantPillBar(
                            state = state,
                            onClose = { shutdown() },
                            onInterrupt = { assistantManager.interrupt() },
                            onCameraToggle = { handleCameraToggle() },
                            cameraCaptureManager = cameraCaptureManager,
                            modifier = Modifier.onGloballyPositioned { coordinates ->
                                wrapper.contentTop = coordinates.positionInWindow().y.toInt()
                            }
                        )
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
