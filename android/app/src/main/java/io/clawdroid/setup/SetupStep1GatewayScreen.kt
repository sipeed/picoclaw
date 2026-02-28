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
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary

@Composable
fun SetupStep1GatewayScreen(viewModel: SetupViewModel) {
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

        Text(
            "Step 1 of 4",
            style = MaterialTheme.typography.labelMedium,
            color = TextSecondary,
        )
        Text(
            "Gateway Connection",
            style = MaterialTheme.typography.headlineMedium,
            color = TextPrimary,
        )
        Text(
            "Configure the HTTP gateway that connects this app to the backend.",
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
        )

        Spacer(Modifier.height(8.dp))

        OutlinedTextField(
            value = uiState.gatewayPort,
            onValueChange = viewModel::onGatewayPortChange,
            label = { Text("Port", color = TextSecondary) },
            placeholder = { Text("18790", color = TextSecondary.copy(alpha = 0.5f)) },
            singleLine = true,
            isError = uiState.gatewayPortError != null,
            supportingText = uiState.gatewayPortError?.let { err -> { Text(err) } },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            colors = setupFieldColors(),
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = uiState.gatewayApiKey,
            onValueChange = viewModel::onGatewayApiKeyChange,
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

        OutlinedButton(
            onClick = viewModel::generateApiKey,
            colors = ButtonDefaults.outlinedButtonColors(contentColor = NeonCyan),
        ) {
            Text("Generate API Key")
        }

        uiState.error?.let { error ->
            Text(
                error,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        Spacer(Modifier.weight(1f))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.End,
        ) {
            Button(
                onClick = viewModel::submitInit,
                enabled = uiState.canProceedStep1,
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
