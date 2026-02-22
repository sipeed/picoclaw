package io.picoclaw.android.feature.chat.component

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil3.compose.AsyncImage
import com.mikepenz.markdown.m3.Markdown
import com.mikepenz.markdown.m3.markdownColor
import com.mikepenz.markdown.m3.markdownTypography
import io.picoclaw.android.core.domain.model.ChatMessage
import io.picoclaw.android.core.domain.model.MessageSender
import io.picoclaw.android.core.ui.theme.AgentBubble
import io.picoclaw.android.core.ui.theme.AgentBubbleBorder
import io.picoclaw.android.core.ui.theme.TextPrimary
import io.picoclaw.android.core.ui.theme.UserBubble
import io.picoclaw.android.core.ui.theme.UserBubbleBorder
import java.io.File

@Composable
fun MessageBubble(
    message: ChatMessage,
    modifier: Modifier = Modifier
) {
    val isUser = message.sender == MessageSender.USER
    val alignment = if (isUser) Alignment.CenterEnd else Alignment.CenterStart
    val bubbleColor = if (isUser) UserBubble else AgentBubble
    val borderColor = if (isUser) UserBubbleBorder else AgentBubbleBorder
    val shape = RoundedCornerShape(
        topStart = 20.dp,
        topEnd = 20.dp,
        bottomStart = if (isUser) 20.dp else 4.dp,
        bottomEnd = if (isUser) 4.dp else 20.dp
    )

    Box(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 3.dp),
        contentAlignment = alignment
    ) {
        Surface(
            shape = shape,
            color = bubbleColor,
            border = BorderStroke(0.5.dp, borderColor),
            modifier = Modifier.widthIn(max = 300.dp)
        ) {
            Column(modifier = Modifier.padding(12.dp)) {
                message.images.forEach { imageData ->
                    val ratio = if (imageData.width > 0 && imageData.height > 0) {
                        imageData.width.toFloat() / imageData.height.toFloat()
                    } else 1f

                    AsyncImage(
                        model = File(imageData.path),
                        contentDescription = null,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(bottom = 8.dp)
                            .heightIn(max = 360.dp)
                            .aspectRatio(ratio, matchHeightConstraintsFirst = ratio < 0.75f),
                        contentScale = ContentScale.Fit
                    )
                }
                if (message.content.isNotEmpty()) {
                    val baseStyle = MaterialTheme.typography.bodyLarge.copy(color = TextPrimary)
                    Markdown(
                        content = message.content,
                        colors = markdownColor(text = TextPrimary),
                        typography = markdownTypography(
                            h1 = baseStyle.copy(fontSize = 20.sp, fontWeight = FontWeight.Bold, lineHeight = 28.sp),
                            h2 = baseStyle.copy(fontSize = 18.sp, fontWeight = FontWeight.Bold, lineHeight = 26.sp),
                            h3 = baseStyle.copy(fontSize = 17.sp, fontWeight = FontWeight.SemiBold, lineHeight = 24.sp),
                            h4 = baseStyle.copy(fontSize = 16.sp, fontWeight = FontWeight.SemiBold, lineHeight = 24.sp),
                            h5 = baseStyle.copy(fontSize = 15.sp, fontWeight = FontWeight.Medium, lineHeight = 22.sp),
                            h6 = baseStyle.copy(fontSize = 14.sp, fontWeight = FontWeight.Medium, lineHeight = 20.sp),
                            paragraph = baseStyle,
                        ),
                    )
                }
            }
        }
    }
}
