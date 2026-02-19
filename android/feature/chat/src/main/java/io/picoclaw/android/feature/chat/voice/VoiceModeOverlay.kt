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
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import io.picoclaw.android.core.domain.model.VoicePhase
import io.picoclaw.android.core.ui.theme.DeepBlack
import io.picoclaw.android.core.ui.theme.GradientCyan
import io.picoclaw.android.core.ui.theme.GradientPurple
import io.picoclaw.android.core.ui.theme.TextPrimary
import io.picoclaw.android.core.ui.theme.TextSecondary
import com.composables.icons.lucide.R as LucideR

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
                .background(DeepBlack)
                .drawBehind {
                    drawCircle(
                        brush = Brush.radialGradient(
                            colors = listOf(
                                GradientCyan.copy(alpha = 0.1f),
                                Color.Transparent
                            ),
                            center = Offset(size.width * 0.3f, size.height * 0.3f),
                            radius = size.width * 0.6f
                        )
                    )
                    drawCircle(
                        brush = Brush.radialGradient(
                            colors = listOf(
                                GradientPurple.copy(alpha = 0.1f),
                                Color.Transparent
                            ),
                            center = Offset(size.width * 0.7f, size.height * 0.7f),
                            radius = size.width * 0.5f
                        )
                    )
                }
        ) {
            IconButton(
                onClick = onClose,
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(16.dp)
            ) {
                Icon(
                    painter = painterResource(LucideR.drawable.lucide_ic_x),
                    contentDescription = "Close",
                    modifier = Modifier.size(28.dp),
                    tint = TextSecondary
                )
            }

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

                Text(
                    text = state.statusText.takeIf { state.phase == VoicePhase.THINKING }
                        ?: phaseLabel(state.phase),
                    style = MaterialTheme.typography.titleMedium,
                    color = TextSecondary
                )

                Spacer(modifier = Modifier.height(24.dp))

                if (state.recognizedText.isNotEmpty()) {
                    Text(
                        text = state.recognizedText,
                        style = MaterialTheme.typography.bodyLarge,
                        color = TextPrimary,
                        textAlign = TextAlign.Center
                    )
                }

                if (state.responseText.isNotEmpty() && state.phase == VoicePhase.SPEAKING) {
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = state.responseText,
                        style = MaterialTheme.typography.bodyMedium,
                        color = TextSecondary,
                        textAlign = TextAlign.Center
                    )
                }

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
    VoicePhase.LISTENING -> "Listening..."
    VoicePhase.SENDING -> "Sending..."
    VoicePhase.THINKING -> "Thinking..."
    VoicePhase.SPEAKING -> "Speaking..."
    VoicePhase.ERROR -> "Error"
}
