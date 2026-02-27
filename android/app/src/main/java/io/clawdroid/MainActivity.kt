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
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import io.clawdroid.backend.config.ConfigSectionDetailScreen
import io.clawdroid.backend.config.ConfigSectionListScreen
import io.clawdroid.core.ui.theme.ClawDroidTheme
import io.clawdroid.feature.chat.screen.ChatScreen
import io.clawdroid.feature.chat.screen.SettingsScreen
import io.clawdroid.navigation.NavRoutes
import io.clawdroid.settings.AppSettingsScreen

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
                            onNavigateBack = { navController.popBackStack() },
                            onNavigateToBackendSettings = { navController.navigate(NavRoutes.BACKEND_SETTINGS) },
                            onNavigateToAppSettings = { navController.navigate(NavRoutes.APP_SETTINGS) },
                        )
                    }
                    composable(NavRoutes.BACKEND_SETTINGS) {
                        ConfigSectionListScreen(
                            onNavigateBack = { navController.popBackStack() },
                            onSectionSelected = { sectionKey ->
                                navController.navigate("backend_settings/$sectionKey")
                            },
                        )
                    }
                    composable(
                        NavRoutes.BACKEND_SETTINGS_SECTION,
                        arguments = listOf(navArgument("sectionKey") { type = NavType.StringType }),
                    ) { backStackEntry ->
                        ConfigSectionDetailScreen(
                            sectionKey = backStackEntry.arguments?.getString("sectionKey") ?: "",
                            onNavigateBack = { navController.popBackStack() },
                        )
                    }
                    composable(NavRoutes.APP_SETTINGS) {
                        AppSettingsScreen(
                            onNavigateBack = { navController.popBackStack() },
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
