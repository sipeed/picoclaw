package io.picoclaw.android.feature.chat.component

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import io.picoclaw.android.core.ui.theme.DeepBlack
import io.picoclaw.android.core.ui.theme.GlassBorder
import io.picoclaw.android.core.ui.theme.GlassWhite
import io.picoclaw.android.core.ui.theme.NeonCyan
import io.picoclaw.android.core.ui.theme.TextPrimary
import io.picoclaw.android.core.ui.theme.TextSecondary
import io.picoclaw.android.core.ui.theme.TextTertiary
import com.composables.icons.lucide.R as LucideR

@Composable
fun MessageInput(
    text: String,
    onTextChanged: (String) -> Unit,
    onSendClick: () -> Unit,
    onCameraClick: () -> Unit,
    onGalleryClick: () -> Unit,
    onMicClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 8.dp)
            .background(
                color = GlassWhite,
                shape = RoundedCornerShape(28.dp)
            )
            .border(
                width = 0.5.dp,
                color = GlassBorder,
                shape = RoundedCornerShape(28.dp)
            )
            .padding(start = 4.dp, end = 4.dp, top = 4.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        IconButton(onClick = onCameraClick) {
            Icon(
                painter = painterResource(LucideR.drawable.lucide_ic_camera),
                contentDescription = "Camera",
                tint = TextSecondary
            )
        }
        IconButton(onClick = onGalleryClick) {
            Icon(
                painter = painterResource(LucideR.drawable.lucide_ic_image),
                contentDescription = "Gallery",
                tint = TextSecondary
            )
        }
        IconButton(onClick = onMicClick) {
            Icon(
                painter = painterResource(LucideR.drawable.lucide_ic_mic),
                contentDescription = "Voice",
                tint = TextSecondary
            )
        }
        OutlinedTextField(
            value = text,
            onValueChange = onTextChanged,
            modifier = Modifier.weight(1f),
            placeholder = { Text("Message...", color = TextTertiary) },
            maxLines = 4,
            shape = RoundedCornerShape(24.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = Color.Transparent,
                unfocusedBorderColor = Color.Transparent,
                focusedContainerColor = Color.Transparent,
                unfocusedContainerColor = Color.Transparent,
                cursorColor = NeonCyan,
                focusedTextColor = TextPrimary,
                unfocusedTextColor = TextPrimary
            )
        )
        Spacer(modifier = Modifier.width(4.dp))
        FilledIconButton(
            onClick = onSendClick,
            shape = CircleShape,
            colors = IconButtonDefaults.filledIconButtonColors(
                containerColor = NeonCyan,
                contentColor = DeepBlack
            )
        ) {
            Icon(
                painter = painterResource(LucideR.drawable.lucide_ic_send_horizontal),
                contentDescription = "Send"
            )
        }
    }
}
