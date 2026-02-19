package io.picoclaw.android.feature.chat.screen

import android.Manifest
import android.content.pm.PackageManager
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.FileProvider
import io.picoclaw.android.core.domain.model.ImageAttachment
import io.picoclaw.android.feature.chat.ChatEvent
import io.picoclaw.android.feature.chat.ChatViewModel
import io.picoclaw.android.feature.chat.component.ConnectionBanner
import io.picoclaw.android.feature.chat.component.ImagePreviewRow
import io.picoclaw.android.feature.chat.component.MessageInput
import io.picoclaw.android.feature.chat.component.MessageList
import io.picoclaw.android.feature.chat.component.StatusIndicator
import io.picoclaw.android.feature.chat.voice.VoiceModeOverlay
import org.koin.androidx.compose.koinViewModel
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

    var cameraImageUri by remember { mutableStateOf<Uri?>(null) }

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

    // RECORD_AUDIO permission
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

    // Back handler for voice mode
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

    LaunchedEffect(uiState.messages.size) {
        if (isNearBottom) {
            listState.animateScrollToItem(0)
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("PicoClaw") },
                    actions = {
                        IconButton(onClick = onNavigateToSettings) {
                            Icon(Icons.Default.Settings, contentDescription = "Settings")
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
            onInterrupt = { viewModel.onEvent(ChatEvent.OnVoiceModeInterrupt) }
        )
    }
}
