package io.picoclaw.android.feature.chat.screen

import android.graphics.BitmapFactory
import android.net.Uri
import android.util.Base64
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.ExperimentalMaterial3Api
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
import org.koin.androidx.compose.koinViewModel
import java.io.ByteArrayOutputStream
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(
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
                uriToBase64(context, uri)?.let { base64 ->
                    viewModel.onEvent(
                        ChatEvent.OnImageAdded(
                            ImageAttachment(
                                uri = uri.toString(),
                                base64 = base64
                            )
                        )
                    )
                }
            }
        }
    }

    val galleryLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent()
    ) { uri ->
        uri?.let {
            uriToBase64(context, it)?.let { base64 ->
                val mimeType = context.contentResolver.getType(it) ?: "image/png"
                viewModel.onEvent(
                    ChatEvent.OnImageAdded(
                        ImageAttachment(
                            uri = it.toString(),
                            base64 = base64,
                            mimeType = mimeType
                        )
                    )
                )
            }
        }
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

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("PicoClaw") })
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

            ImagePreviewRow(
                images = uiState.pendingImages,
                onRemove = { viewModel.onEvent(ChatEvent.OnImageRemoved(it)) }
            )

            MessageInput(
                text = uiState.inputText,
                onTextChanged = { viewModel.onEvent(ChatEvent.OnInputChanged(it)) },
                onSendClick = { viewModel.onEvent(ChatEvent.OnSendClick) },
                onCameraClick = {
                    val imagesDir = File(context.cacheDir, "images").apply { mkdirs() }
                    val imageFile = File(imagesDir, "camera_${System.currentTimeMillis()}.jpg")
                    val uri = FileProvider.getUriForFile(
                        context,
                        "${context.packageName}.fileprovider",
                        imageFile
                    )
                    cameraImageUri = uri
                    cameraLauncher.launch(uri)
                },
                onGalleryClick = {
                    galleryLauncher.launch("image/*")
                }
            )
        }
    }
}

private fun uriToBase64(context: android.content.Context, uri: Uri): String? {
    return try {
        context.contentResolver.openInputStream(uri)?.use { input ->
            val bitmap = BitmapFactory.decodeStream(input)
            val output = ByteArrayOutputStream()
            bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG, 100, output)
            Base64.encodeToString(output.toByteArray(), Base64.NO_WRAP)
        }
    } catch (_: Exception) {
        null
    }
}
