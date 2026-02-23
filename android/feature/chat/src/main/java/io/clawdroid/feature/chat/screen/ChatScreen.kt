package io.clawdroid.feature.chat.screen

import android.Manifest
import android.content.pm.PackageManager
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.core.content.FileProvider
import io.clawdroid.core.domain.model.ImageAttachment
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.GradientCyan
import io.clawdroid.core.ui.theme.GradientPink
import io.clawdroid.core.ui.theme.GradientPurple
import io.clawdroid.core.ui.theme.TextSecondary
import io.clawdroid.feature.chat.ChatEvent
import io.clawdroid.feature.chat.ChatViewModel
import com.composables.icons.lucide.R as LucideR
import io.clawdroid.feature.chat.component.ConnectionBanner
import io.clawdroid.feature.chat.component.ImagePreviewRow
import io.clawdroid.feature.chat.component.MessageInput
import io.clawdroid.feature.chat.component.MessageList
import io.clawdroid.feature.chat.component.StatusIndicator
import io.clawdroid.feature.chat.voice.CameraCaptureManager
import io.clawdroid.feature.chat.voice.VoiceModeOverlay
import org.koin.androidx.compose.koinViewModel
import org.koin.compose.koinInject
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(
    onNavigateToSettings: () -> Unit = {},
    viewModel: ChatViewModel = koinViewModel()
) {
    val context = LocalContext.current
    val uiState by viewModel.uiState.collectAsState()
    val listState = rememberLazyListState()
    val cameraCaptureManager: CameraCaptureManager = koinInject()

    var cameraImageUri by remember { mutableStateOf<Uri?>(null) }

    val voiceCameraPermissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { granted ->
        if (granted) {
            viewModel.onEvent(ChatEvent.OnVoiceCameraToggle)
        }
    }

    val onVoiceCameraToggle = {
        if (uiState.voiceModeState.isCameraActive) {
            viewModel.onEvent(ChatEvent.OnVoiceCameraToggle)
        } else {
            if (ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA)
                == PackageManager.PERMISSION_GRANTED
            ) {
                viewModel.onEvent(ChatEvent.OnVoiceCameraToggle)
            } else {
                voiceCameraPermissionLauncher.launch(Manifest.permission.CAMERA)
            }
        }
    }

    val cameraLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.TakePicture()
    ) { success ->
        if (success) {
            cameraImageUri?.let { uri ->
                viewModel.onEvent(
                    ChatEvent.OnImageAdded(ImageAttachment(uri = uri.toString()))
                )
            }
        }
    }

    val launchCamera = {
        val imagesDir = File(context.cacheDir, "images").apply { mkdirs() }
        val imageFile = File(imagesDir, "camera_${System.currentTimeMillis()}.jpg")
        val uri = FileProvider.getUriForFile(
            context,
            "${context.packageName}.fileprovider",
            imageFile
        )
        cameraImageUri = uri
        cameraLauncher.launch(uri)
    }

    val cameraPermissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { granted ->
        if (granted) launchCamera()
    }

    val galleryLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent()
    ) { uri ->
        uri?.let {
            val mimeType = context.contentResolver.getType(it) ?: "image/jpeg"
            viewModel.onEvent(
                ChatEvent.OnImageAdded(ImageAttachment(uri = it.toString(), mimeType = mimeType))
            )
        }
    }

    var pendingVoiceStart by remember { mutableStateOf(false) }

    val micPermissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { granted ->
        if (granted) {
            viewModel.onEvent(ChatEvent.OnVoiceModeStart)
        }
        pendingVoiceStart = false
    }

    val onMicClick = {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.RECORD_AUDIO)
            == PackageManager.PERMISSION_GRANTED
        ) {
            viewModel.onEvent(ChatEvent.OnVoiceModeStart)
        } else {
            pendingVoiceStart = true
            micPermissionLauncher.launch(Manifest.permission.RECORD_AUDIO)
        }
    }

    BackHandler(enabled = uiState.voiceModeState.isActive) {
        viewModel.onEvent(ChatEvent.OnVoiceModeStop)
    }

    val shouldLoadMore by remember {
        derivedStateOf {
            val lastVisibleItem = listState.layoutInfo.visibleItemsInfo.lastOrNull()
            lastVisibleItem != null &&
                lastVisibleItem.index >= listState.layoutInfo.totalItemsCount - 5
        }
    }

    LaunchedEffect(shouldLoadMore) {
        if (shouldLoadMore) viewModel.onEvent(ChatEvent.OnLoadMore)
    }

    LaunchedEffect(Unit) {
        viewModel.scrollToBottom.collect {
            listState.animateScrollToItem(0)
        }
    }

    val isNearBottom by remember {
        derivedStateOf { listState.firstVisibleItemIndex <= 3 }
    }

    LaunchedEffect(uiState.messages.firstOrNull()?.id) {
        if (isNearBottom) {
            listState.animateScrollToItem(0)
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(DeepBlack)
            .drawBehind {
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientCyan.copy(alpha = 0.07f),
                            Color.Transparent
                        ),
                        center = Offset(size.width * 0.15f, size.height * 0.1f),
                        radius = size.width * 0.8f
                    )
                )
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientPurple.copy(alpha = 0.07f),
                            Color.Transparent
                        ),
                        center = Offset(size.width * 0.85f, size.height * 0.9f),
                        radius = size.width * 0.7f
                    )
                )
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientPink.copy(alpha = 0.03f),
                            Color.Transparent
                        ),
                        center = Offset(size.width * 0.5f, size.height * 0.5f),
                        radius = size.width * 0.5f
                    )
                )
            }
    ) {
        Scaffold(
            containerColor = Color.Transparent,
            topBar = {
                TopAppBar(
                    title = { Text("ClawDroid") },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = Color.Transparent
                    ),
                    actions = {
                        IconButton(onClick = onNavigateToSettings) {
                            Icon(
                                painter = painterResource(LucideR.drawable.lucide_ic_settings),
                                contentDescription = "Settings",
                                tint = TextSecondary
                            )
                        }
                    }
                )
            }
        ) { padding ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
            ) {
                ConnectionBanner(connectionState = uiState.connectionState)

                MessageList(
                    messages = uiState.messages,
                    listState = listState,
                    isLoadingMore = uiState.isLoadingMore,
                    modifier = Modifier.weight(1f)
                )

                StatusIndicator(label = uiState.statusLabel)

                ImagePreviewRow(
                    images = uiState.pendingImages,
                    onRemove = { viewModel.onEvent(ChatEvent.OnImageRemoved(it)) }
                )

                MessageInput(
                    text = uiState.inputText,
                    onTextChanged = { viewModel.onEvent(ChatEvent.OnInputChanged(it)) },
                    onSendClick = { viewModel.onEvent(ChatEvent.OnSendClick) },
                    onCameraClick = {
                        if (ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA)
                            == PackageManager.PERMISSION_GRANTED
                        ) {
                            launchCamera()
                        } else {
                            cameraPermissionLauncher.launch(Manifest.permission.CAMERA)
                        }
                    },
                    onGalleryClick = {
                        galleryLauncher.launch("image/*")
                    },
                    onMicClick = onMicClick
                )
            }
        }

        VoiceModeOverlay(
            state = uiState.voiceModeState,
            onClose = { viewModel.onEvent(ChatEvent.OnVoiceModeStop) },
            onInterrupt = { viewModel.onEvent(ChatEvent.OnVoiceModeInterrupt) },
            onCameraToggle = onVoiceCameraToggle,
            cameraCaptureManager = cameraCaptureManager
        )
    }
}
