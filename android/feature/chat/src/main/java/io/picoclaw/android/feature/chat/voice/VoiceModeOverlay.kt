package io.picoclaw.android.feature.chat.voice

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import io.picoclaw.android.core.domain.model.VoicePhase

@Composable
fun VoiceModeOverlay(
    state: VoiceModeState,
    onClose: () -> Unit,
    onInterrupt: () -> Unit,
    modifier: Modifier = Modifier
) {
    AnimatedVisibility(
        visible = state.isActive,
        enter = fadeIn() + slideInVertically { it / 2 },
        exit = fadeOut() + slideOutVertically { it / 2 }
    ) {
        Box(
            modifier = modifier
                .fillMaxSize()
                .background(MaterialTheme.colorScheme.surface)
        ) {
            // Close button
            IconButton(
                onClick = onClose,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(16.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "閉じる",
                    modifier = Modifier.size(28.dp)
                )
            }

            // Center content
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center
            ) {
                val interruptable = state.phase != VoicePhase.LISTENING &&
                    state.phase != VoicePhase.IDLE

                VoiceOrb(
                    phase = state.phase,
                    amplitudeNormalized = state.amplitudeNormalized,
                    modifier = if (interruptable) {
                        Modifier.clickable(
                            indication = null,
                            interactionSource = remember { MutableInteractionSource() }
                        ) { onInterrupt() }
                    } else {
                        Modifier
                    }
                )

                Spacer(modifier = Modifier.height(32.dp))

                // Phase label
                Text(
                    text = state.statusText.takeIf { state.phase == VoicePhase.THINKING }
                        ?: phaseLabel(state.phase),
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )

                Spacer(modifier = Modifier.height(24.dp))

                // Recognized text
                if (state.recognizedText.isNotEmpty()) {
                    Text(
                        text = state.recognizedText,
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurface,
                        textAlign = TextAlign.Center
                    )
                }

                // Response text
                if (state.responseText.isNotEmpty() && state.phase == VoicePhase.SPEAKING) {
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = state.responseText,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f),
                        textAlign = TextAlign.Center
                    )
                }

                // Error message
                if (state.errorMessage != null) {
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = state.errorMessage,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.error,
                        textAlign = TextAlign.Center
                    )
                }
            }
        }
    }
}

private fun phaseLabel(phase: VoicePhase): String = when (phase) {
    VoicePhase.IDLE -> ""
    VoicePhase.LISTENING -> "聞き取り中..."
    VoicePhase.SENDING -> "送信中..."
    VoicePhase.THINKING -> "考え中..."
    VoicePhase.SPEAKING -> "話しています..."
    VoicePhase.ERROR -> "エラー"
}
