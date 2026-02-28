package io.clawdroid.setup

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary

@Composable
fun SetupStep2LlmScreen(viewModel: SetupViewModel) {
    val uiState by viewModel.uiState.collectAsState()
    var apiKeyHidden by remember { mutableStateOf(true) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Spacer(Modifier.height(32.dp))

        Text("Step 2 of 4", style = MaterialTheme.typography.labelMedium, color = TextSecondary)
        Text("LLM Settings", style = MaterialTheme.typography.headlineMedium, color = TextPrimary)
        Text(
            "Configure the language model used by the agent.",
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
        )

        Spacer(Modifier.height(8.dp))

        OutlinedTextField(
            value = uiState.llmModel,
            onValueChange = viewModel::onLlmModelChange,
            label = { Text("Model", color = TextSecondary) },
            placeholder = { Text("e.g. openai/gpt-4o", color = TextSecondary.copy(alpha = 0.5f)) },
            singleLine = true,
            colors = setupFieldColors(),
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = uiState.llmApiKey,
            onValueChange = viewModel::onLlmApiKeyChange,
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
            colors = setupFieldColors(),
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = uiState.llmBaseUrl,
            onValueChange = viewModel::onLlmBaseUrlChange,
            label = { Text("Base URL", color = TextSecondary) },
            placeholder = { Text("https://openrouter.ai/api/v1", color = TextSecondary.copy(alpha = 0.5f)) },
            singleLine = true,
            colors = setupFieldColors(),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.weight(1f))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            TextButton(onClick = { viewModel.skipStep(2) }) {
                Text("Set up later", color = TextSecondary)
            }
            Button(
                onClick = { viewModel.nextStep(2) },
                colors = ButtonDefaults.buttonColors(
                    containerColor = NeonCyan,
                    contentColor = DeepBlack,
                ),
            ) {
                Text("Next")
            }
        }
    }
}
