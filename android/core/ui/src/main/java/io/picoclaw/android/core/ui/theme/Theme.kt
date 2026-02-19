package io.picoclaw.android.core.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp

private val FuturisticDarkScheme = darkColorScheme(
    primary = NeonCyan,
    onPrimary = DeepBlack,
    primaryContainer = NeonCyan.copy(alpha = 0.15f),
    onPrimaryContainer = NeonCyan,
    secondary = ElectricPurple,
    onSecondary = DeepBlack,
    secondaryContainer = ElectricPurple.copy(alpha = 0.15f),
    onSecondaryContainer = ElectricPurple,
    tertiary = AccentPink,
    onTertiary = DeepBlack,
    background = DeepBlack,
    onBackground = TextPrimary,
    surface = DarkSurface,
    onSurface = TextPrimary,
    surfaceVariant = DarkCard,
    onSurfaceVariant = TextSecondary,
    outline = GlassBorder,
    outlineVariant = GlassWhite,
    error = DisconnectedRed,
    onError = TextPrimary,
)

private val FuturisticShapes = Shapes(
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(24.dp),
    extraLarge = RoundedCornerShape(28.dp),
)

@Composable
fun PicoClawTheme(
    content: @Composable () -> Unit
) {
    MaterialTheme(
        colorScheme = FuturisticDarkScheme,
        typography = Typography,
        shapes = FuturisticShapes,
        content = content
    )
}
