package io.clawdroid.settings

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.composables.icons.lucide.R as LucideR
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.GlassBorder
import io.clawdroid.core.ui.theme.GlassWhite
import io.clawdroid.core.ui.theme.GradientCyan
import io.clawdroid.core.ui.theme.GradientPurple
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary
import org.koin.compose.viewmodel.koinViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AppSettingsScreen(
    onNavigateBack: () -> Unit,
    viewModel: AppSettingsViewModel = koinViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    var apiKeyHidden by remember { mutableStateOf(true) }
    val saveEnabled by remember { derivedStateOf { !uiState.hasErrors && !uiState.saving } }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(DeepBlack)
            .drawBehind {
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientCyan.copy(alpha = 0.07f),
                            Color.Transparent,
                        ),
                        center = Offset(size.width * 0.15f, size.height * 0.1f),
                        radius = size.width * 0.8f,
                    ),
                )
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientPurple.copy(alpha = 0.07f),
                            Color.Transparent,
                        ),
                        center = Offset(size.width * 0.85f, size.height * 0.9f),
                        radius = size.width * 0.7f,
                    ),
                )
            },
    ) {
        Scaffold(
            containerColor = Color.Transparent,
            topBar = {
                TopAppBar(
                    title = { Text("Connection") },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = Color.Transparent,
                    ),
                    navigationIcon = {
                        IconButton(onClick = onNavigateBack) {
                            Icon(
                                painter = painterResource(LucideR.drawable.lucide_ic_arrow_left),
                                contentDescription = "Back",
                                tint = TextSecondary,
                            )
                        }
                    },
                    actions = {
                        Button(
                            onClick = { viewModel.save(onNavigateBack) },
                            enabled = saveEnabled,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = NeonCyan,
                                contentColor = DeepBlack,
                            ),
                            modifier = Modifier.padding(end = 8.dp),
                        ) {
                            Text(if (uiState.saving) "Saving…" else "Save")
                        }
                    },
                )
            },
        ) { padding ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(horizontal = 16.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text(
                    "Gateway Connection",
                    style = MaterialTheme.typography.titleMedium,
                    color = NeonCyan,
                )

                OutlinedTextField(
                    value = uiState.httpPort,
                    onValueChange = { viewModel.onHttpPortChange(it) },
                    label = { Text("Port", color = TextSecondary) },
                    placeholder = { Text("18790", color = TextSecondary.copy(alpha = 0.5f)) },
                    singleLine = true,
                    isError = uiState.httpPortError != null,
                    supportingText = uiState.httpPortError?.let { err -> { Text(err) } },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    colors = appSettingsFieldColors(),
                    modifier = Modifier.fillMaxWidth(),
                )

                OutlinedTextField(
                    value = uiState.apiKey,
                    onValueChange = { viewModel.onApiKeyChange(it) },
                    label = { Text("API Key", color = TextSecondary) },
                    singleLine = true,
                    visualTransformation = if (apiKeyHidden) PasswordVisualTransformation() else VisualTransformation.None,
                    trailingIcon = {
                        TextButton(onClick = { apiKeyHidden = !apiKeyHidden }) {
                            Text(
                                if (apiKeyHidden) "Show" else "Hide",
                                color = NeonCyan,
                                style = MaterialTheme.typography.labelSmall,
                            )
                        }
                    },
                    colors = appSettingsFieldColors(),
                    modifier = Modifier.fillMaxWidth(),
                )

                uiState.error?.let { error ->
                    Text(
                        error,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        }
    }
}

@Composable
private fun appSettingsFieldColors() = OutlinedTextFieldDefaults.colors(
    focusedBorderColor = NeonCyan.copy(alpha = 0.5f),
    unfocusedBorderColor = GlassBorder,
    focusedContainerColor = GlassWhite,
    unfocusedContainerColor = Color.Transparent,
    focusedTextColor = TextPrimary,
    unfocusedTextColor = TextPrimary,
)
