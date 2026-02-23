package io.clawdroid.feature.chat.voice

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.dp
import io.clawdroid.core.domain.model.VoicePhase
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.sin

private val ListeningColor = Color(0xFF00D4FF)
private val SendingColor = Color(0xFFFF8C42)
private val ThinkingColor = Color(0xFFA855F7)
private val SpeakingColor = Color(0xFF22C55E)
private val ErrorColor = Color(0xFFEF4444)
private val IdleColor = Color(0xFF4A5568)

@Composable
fun VoiceOrb(
    phase: VoicePhase,
    amplitudeNormalized: Float,
    modifier: Modifier = Modifier
) {
    val transition = rememberInfiniteTransition(label = "orb")

    val pulse by transition.animateFloat(
        initialValue = 0.95f,
        targetValue = 1.05f,
        animationSpec = infiniteRepeatable(
            animation = tween(
                durationMillis = when (phase) {
                    VoicePhase.THINKING -> 800
                    VoicePhase.SPEAKING -> 600
                    else -> 1200
                },
                easing = LinearEasing
            ),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse"
    )

    val rotation by transition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 4000, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "rotation"
    )

    val glowAlpha by transition.animateFloat(
        initialValue = 0.3f,
        targetValue = 0.6f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "glow"
    )

    val primaryColor = when (phase) {
        VoicePhase.LISTENING -> ListeningColor
        VoicePhase.SENDING -> SendingColor
        VoicePhase.THINKING -> ThinkingColor
        VoicePhase.SPEAKING -> SpeakingColor
        VoicePhase.ERROR -> ErrorColor
        VoicePhase.IDLE -> IdleColor
    }

    Canvas(modifier = modifier.size(200.dp)) {
        val center = Offset(size.width / 2f, size.height / 2f)
        val baseRadius = size.minDimension / 4f

        val scaleMultiplier = if (phase == VoicePhase.LISTENING) {
            1f + amplitudeNormalized * 0.3f
        } else {
            pulse
        }

        val orbRadius = baseRadius * scaleMultiplier

        // Glow
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(
                    primaryColor.copy(alpha = glowAlpha),
                    primaryColor.copy(alpha = 0f)
                ),
                center = center,
                radius = orbRadius * 2f
            ),
            radius = orbRadius * 2f,
            center = center
        )

        // Main orb
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(
                    primaryColor,
                    primaryColor.copy(alpha = 0.7f)
                ),
                center = Offset(
                    center.x - orbRadius * 0.2f,
                    center.y - orbRadius * 0.2f
                ),
                radius = orbRadius * 1.5f
            ),
            radius = orbRadius,
            center = center
        )

        // Wave rings
        drawWaveRings(center, orbRadius, primaryColor, rotation, phase, amplitudeNormalized)
    }
}

private fun DrawScope.drawWaveRings(
    center: Offset,
    orbRadius: Float,
    color: Color,
    rotation: Float,
    phase: VoicePhase,
    amplitude: Float
) {
    val ringCount = 3
    for (i in 1..ringCount) {
        val ringRadius = orbRadius * (1.2f + i * 0.25f)
        val alpha = (0.4f - i * 0.1f).coerceAtLeast(0.05f)

        val waveAmplitude = when (phase) {
            VoicePhase.LISTENING -> amplitude * 8f
            VoicePhase.SPEAKING -> 4f
            VoicePhase.THINKING -> 2f
            else -> 0f
        }

        if (waveAmplitude > 0f) {
            val path = androidx.compose.ui.graphics.Path()
            val steps = 72
            for (step in 0..steps) {
                val angle = (step.toFloat() / steps) * 2f * PI.toFloat()
                val wave = sin(angle * 6f + Math.toRadians(rotation.toDouble()).toFloat() * (i + 1)) * waveAmplitude
                val r = ringRadius + wave
                val x = center.x + cos(angle) * r
                val y = center.y + sin(angle) * r
                if (step == 0) path.moveTo(x, y) else path.lineTo(x, y)
            }
            path.close()
            drawPath(
                path = path,
                color = color.copy(alpha = alpha),
                style = Stroke(width = 2f)
            )
        } else {
            drawCircle(
                color = color.copy(alpha = alpha),
                radius = ringRadius,
                center = center,
                style = Stroke(width = 1.5f)
            )
        }
    }
}
