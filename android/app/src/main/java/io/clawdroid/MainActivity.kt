package io.clawdroid

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import io.clawdroid.core.ui.theme.ClawDroidTheme
import io.clawdroid.feature.chat.screen.ChatScreen
import io.clawdroid.feature.chat.screen.SettingsScreen
import io.clawdroid.navigation.NavRoutes

class MainActivity : ComponentActivity() {

    private val notificationPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { /* User choice recorded; no further action needed. */ }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        requestNotificationPermissionIfNeeded()
        enableEdgeToEdge()
        setContent {
            ClawDroidTheme {
                val navController = rememberNavController()
                NavHost(navController = navController, startDestination = NavRoutes.CHAT) {
                    composable(NavRoutes.CHAT) {
                        ChatScreen(
                            onNavigateToSettings = { navController.navigate(NavRoutes.SETTINGS) }
                        )
                    }
                    composable(NavRoutes.SETTINGS) {
                        SettingsScreen(
                            onNavigateBack = { navController.popBackStack() }
                        )
                    }
                }
            }
        }
    }

    private fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
                != PackageManager.PERMISSION_GRANTED
            ) {
                notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        }
    }
}
