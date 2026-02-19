package io.picoclaw.android.feature.chat.component

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import io.picoclaw.android.core.domain.model.ConnectionState
import io.picoclaw.android.core.ui.theme.DisconnectedRed
import io.picoclaw.android.core.ui.theme.ReconnectingYellow

@Composable
fun ConnectionBanner(
    connectionState: ConnectionState,
    modifier: Modifier = Modifier
) {
    AnimatedVisibility(visible = connectionState != ConnectionState.CONNECTED) {
        val (accentColor, text) = when (connectionState) {
            ConnectionState.CONNECTING -> ReconnectingYellow to "Connecting..."
            ConnectionState.RECONNECTING -> ReconnectingYellow to "Reconnecting..."
            ConnectionState.DISCONNECTED -> DisconnectedRed to "Disconnected"
            ConnectionState.CONNECTED -> Color.Transparent to ""
        }
        Box(
            modifier = modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 4.dp)
                .background(
                    color = accentColor.copy(alpha = 0.1f),
                    shape = RoundedCornerShape(12.dp)
                )
                .border(
                    width = 0.5.dp,
                    color = accentColor.copy(alpha = 0.3f),
                    shape = RoundedCornerShape(12.dp)
                )
                .padding(vertical = 8.dp),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = text,
                color = accentColor,
                fontSize = 12.sp,
                letterSpacing = 0.5.sp
            )
        }
    }
}
