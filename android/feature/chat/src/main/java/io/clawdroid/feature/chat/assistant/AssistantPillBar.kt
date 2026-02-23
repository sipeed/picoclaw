package io.clawdroid.feature.chat.assistant

import androidx.camera.view.PreviewView
import androidx.camera.view.PreviewView.ImplementationMode
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandVertically
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.compose.LocalLifecycleOwner
import io.clawdroid.core.domain.model.VoicePhase
import io.clawdroid.core.ui.theme.GradientCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary
import io.clawdroid.feature.chat.voice.CameraCaptureManager
import io.clawdroid.feature.chat.voice.ChatTurn
import io.clawdroid.feature.chat.voice.VoiceModeState
import com.composables.icons.lucide.R as LucideR
import kotlin.math.PI
import kotlin.math.sin

private val PillBackground = Color(0xE6141428)
private val ListeningColor = Color(0xFF00D4FF)
private val ThinkingColor = Color(0xFFA855F7)
private val SpeakingColor = Color(0xFF22C55E)
private val ErrorColor = Color(0xFFEF4444)

@Composable
fun AssistantPillBar(
    state: VoiceModeState,
    isAtTop: Boolean,
    onClose: () -> Unit,
    onInterrupt: () -> Unit,
    onPositionChange: (Boolean) -> Unit,
    onCameraToggle: () -> Unit,
    onScreenCaptureToggle: () -> Unit,
    cameraCaptureManager: CameraCaptureManager,
    modifier: Modifier = Modifier
) {
    val isExpanded = state.phase == VoicePhase.THINKING ||
        state.phase == VoicePhase.SPEAKING ||
        state.phase == VoicePhase.ERROR

    val interruptable = state.phase != VoicePhase.LISTENING &&
        state.phase != VoicePhase.IDLE

    var historyExpanded by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 8.dp, vertical = 8.dp)
    ) {
        // Camera preview above the pill bar
        AnimatedVisibility(
            visible = state.isCameraActive,
            enter = expandVertically(expandFrom = if (isAtTop) Alignment.Top else Alignment.Bottom),
            exit = shrinkVertically(shrinkTowards = if (isAtTop) Alignment.Top else Alignment.Bottom)
        ) {
            val lifecycleOwner = LocalLifecycleOwner.current

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 8.dp),
                contentAlignment = Alignment.CenterEnd
            ) {
                Box(
                    modifier = Modifier
                        .width(120.dp)
                        .height(90.dp)
                        .clip(RoundedCornerShape(12.dp))
                ) {
                    AndroidView(
                        factory = { ctx ->
                            PreviewView(ctx).apply {
                                implementationMode = ImplementationMode.COMPATIBLE
                                cameraCaptureManager.bind(lifecycleOwner, this)
                            }
                        },
                        modifier = Modifier.fillMaxSize()
                    )
                }
            }

            DisposableEffect(Unit) {
                onDispose {
                    cameraCaptureManager.unbind()
                }
            }
        }

        // Main pill bar
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(24.dp))
                .background(
                    Brush.verticalGradient(
                        colors = listOf(
                            PillBackground,
                            PillBackground.copy(alpha = 0.95f)
                        )
                    )
                )
                .animateContentSize()
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp)
            ) {
                // Expanded content: response text area
                AnimatedVisibility(
                    visible = state.chatHistory.isNotEmpty() || (isExpanded && (state.responseText.isNotEmpty() || state.errorMessage != null))
                ) {
                    Column(
                        modifier = Modifier
                            .padding(bottom = 12.dp)
                            .clickable(
                                indication = null,
                                interactionSource = remember { MutableInteractionSource() }
                            ) { historyExpanded = !historyExpanded }
                    ) {
                        if (historyExpanded) {
                            Column(
                                modifier = Modifier
                                    .heightIn(max = 240.dp)
                                    .verticalScroll(rememberScrollState()),
                                verticalArrangement = Arrangement.spacedBy(6.dp)
                            ) {
                                for (turn in state.chatHistory) {
                                    ChatBubble(turn)
                                }
                                if (state.responseText.isNotEmpty() &&
                                    (state.chatHistory.isEmpty() || state.chatHistory.last().text != state.responseText)
                                ) {
                                    ChatBubble(ChatTurn("assistant", state.responseText))
                                }
                            }
                        } else {
                            val displayText = state.responseText.ifEmpty {
                                state.chatHistory.lastOrNull { it.role == "assistant" }?.text.orEmpty()
                            }
                            if (displayText.isNotEmpty()) {
                                Text(
                                    text = displayText,
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = TextPrimary,
                                    maxLines = 6,
                                    overflow = TextOverflow.Ellipsis
                                )
                            }
                        }
                        if (state.errorMessage != null) {
                            Text(
                                text = state.errorMessage,
                                style = MaterialTheme.typography.bodySmall,
                                color = ErrorColor
                            )
                        }
                    }
                }

                // Status / recognized text row
                AnimatedVisibility(
                    visible = isExpanded && (state.recognizedText.isNotEmpty() || state.statusText != null)
                ) {
                    Text(
                        text = state.statusText ?: state.recognizedText,
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.padding(bottom = 8.dp)
                    )
                }

                // Bottom row: waveform + controls
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .then(
                            if (interruptable) {
                                Modifier.clickable(
                                    indication = null,
                                    interactionSource = remember { MutableInteractionSource() }
                                ) { onInterrupt() }
                            } else Modifier
                        ),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    // Mic icon
                    Icon(
                        painter = painterResource(LucideR.drawable.lucide_ic_mic),
                        contentDescription = "Microphone",
                        modifier = Modifier.size(20.dp),
                        tint = phaseColor(state.phase)
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    // Waveform
                    WaveformBar(
                        phase = state.phase,
                        amplitude = state.amplitudeNormalized,
                        modifier = Modifier
                            .weight(1f)
                            .height(28.dp)
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    // Screen capture toggle
                    IconButton(
                        onClick = onScreenCaptureToggle,
                        modifier = Modifier.size(36.dp)
                    ) {
                        Icon(
                            painter = painterResource(
                                if (state.isScreenCaptureActive) LucideR.drawable.lucide_ic_monitor_off
                                else LucideR.drawable.lucide_ic_monitor
                            ),
                            contentDescription = if (state.isScreenCaptureActive) "Turn off screen capture" else "Turn on screen capture",
                            modifier = Modifier.size(18.dp),
                            tint = if (state.isScreenCaptureActive) GradientCyan else TextSecondary
                        )
                    }

                    // Camera toggle
                    IconButton(
                        onClick = onCameraToggle,
                        modifier = Modifier.size(36.dp)
                    ) {
                        Icon(
                            painter = painterResource(
                                if (state.isCameraActive) LucideR.drawable.lucide_ic_camera_off
                                else LucideR.drawable.lucide_ic_camera
                            ),
                            contentDescription = if (state.isCameraActive) "Turn off camera" else "Turn on camera",
                            modifier = Modifier.size(18.dp),
                            tint = if (state.isCameraActive) GradientCyan else TextSecondary
                        )
                    }

                    // Move position toggle
                    IconButton(
                        onClick = { onPositionChange(!isAtTop) },
                        modifier = Modifier.size(36.dp)
                    ) {
                        Icon(
                            painter = painterResource(
                                if (isAtTop) LucideR.drawable.lucide_ic_chevron_down
                                else LucideR.drawable.lucide_ic_chevron_up
                            ),
                            contentDescription = if (isAtTop) "Move to bottom" else "Move to top",
                            modifier = Modifier.size(18.dp),
                            tint = TextSecondary
                        )
                    }

                    // Close button
                    IconButton(
                        onClick = onClose,
                        modifier = Modifier.size(36.dp)
                    ) {
                        Icon(
                            painter = painterResource(LucideR.drawable.lucide_ic_x),
                            contentDescription = "Close",
                            modifier = Modifier.size(18.dp),
                            tint = TextSecondary
                        )
                    }
                }
            }
        }
    }
}

private val UserBubbleBackground = Color(0xFF2563EB)
private val AssistantBubbleBackground = Color(0xFF1E1E3A)

@Composable
private fun ChatBubble(turn: ChatTurn) {
    val isUser = turn.role == "user"
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start
    ) {
        Text(
            text = turn.text,
            style = MaterialTheme.typography.bodySmall,
            color = if (isUser) Color.White else TextPrimary,
            modifier = Modifier
                .widthIn(max = 260.dp)
                .background(
                    color = if (isUser) UserBubbleBackground else AssistantBubbleBackground,
                    shape = RoundedCornerShape(12.dp)
                )
                .padding(horizontal = 10.dp, vertical = 6.dp)
        )
    }
}

@Composable
private fun WaveformBar(
    phase: VoicePhase,
    amplitude: Float,
    modifier: Modifier = Modifier
) {
    val transition = rememberInfiniteTransition(label = "waveform")
    val animPhase by transition.animateFloat(
        initialValue = 0f,
        targetValue = 2f * PI.toFloat(),
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1500, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "wavePhase"
    )

    val color = phaseColor(phase)

    Canvas(modifier = modifier) {
        val barCount = 32
        val barWidth = size.width / barCount
        val centerY = size.height / 2f

        for (i in 0 until barCount) {
            val x = i * barWidth + barWidth / 2f
            val normalizedX = i.toFloat() / barCount

            val barHeight = when (phase) {
                VoicePhase.LISTENING -> {
                    val wave = sin(normalizedX * 4f * PI.toFloat() + animPhase).coerceIn(-1f, 1f)
                    val base = 0.15f
                    val dynamic = amplitude * 0.85f * ((wave + 1f) / 2f)
                    (base + dynamic) * size.height
                }
                VoicePhase.THINKING -> {
                    val wave = sin(normalizedX * 3f * PI.toFloat() + animPhase * 2f)
                    (0.2f + 0.15f * ((wave + 1f) / 2f)) * size.height
                }
                VoicePhase.SPEAKING -> {
                    val wave = sin(normalizedX * 5f * PI.toFloat() + animPhase * 2f)
                    (0.2f + 0.4f * ((wave + 1f) / 2f)) * size.height
                }
                else -> 0.1f * size.height
            }

            drawLine(
                color = color.copy(alpha = 0.6f + 0.4f * (barHeight / size.height)),
                start = Offset(x, centerY - barHeight / 2f),
                end = Offset(x, centerY + barHeight / 2f),
                strokeWidth = barWidth * 0.5f
            )
        }
    }
}

private fun phaseColor(phase: VoicePhase): Color = when (phase) {
    VoicePhase.LISTENING -> ListeningColor
    VoicePhase.SENDING -> Color(0xFFFF8C42)
    VoicePhase.THINKING -> ThinkingColor
    VoicePhase.SPEAKING -> SpeakingColor
    VoicePhase.ERROR -> ErrorColor
    VoicePhase.IDLE -> Color(0xFF4A5568)
}
