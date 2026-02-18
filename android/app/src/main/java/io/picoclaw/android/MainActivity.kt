package io.picoclaw.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.picoclaw.android.core.ui.theme.PicoClawTheme
import io.picoclaw.android.feature.chat.screen.ChatScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            PicoClawTheme {
                ChatScreen()
            }
        }
    }
}
