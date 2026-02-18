package io.picoclaw.android.feature.chat.screen

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
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import io.picoclaw.android.feature.chat.ChatEvent
import io.picoclaw.android.feature.chat.ChatViewModel
import io.picoclaw.android.feature.chat.component.ConnectionBanner
import io.picoclaw.android.feature.chat.component.ImagePreviewRow
import io.picoclaw.android.feature.chat.component.MessageInput
import io.picoclaw.android.feature.chat.component.MessageList
import org.koin.androidx.compose.koinViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(
    viewModel: ChatViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    val listState = rememberLazyListState()

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
                onCameraClick = { /* TODO */ },
                onGalleryClick = { /* TODO */ }
            )
        }
    }
}
