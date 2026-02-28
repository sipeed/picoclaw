package io.clawdroid.setup

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary

@Composable
fun SetupCompleteScreen(
    viewModel: SetupViewModel,
    onSetupComplete: () -> Unit,
) {
    val uiState by viewModel.uiState.collectAsState()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text("Step 4 of 4", style = MaterialTheme.typography.labelMedium, color = TextSecondary)

        Spacer(Modifier.height(16.dp))

        Text(
            "Setup Complete!",
            style = MaterialTheme.typography.headlineMedium,
            color = TextPrimary,
        )

        Spacer(Modifier.height(12.dp))

        Text(
            "You can change these settings later from the Settings screen.",
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
        )

        uiState.error?.let { error ->
            Spacer(Modifier.height(12.dp))
            Text(
                error,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        Spacer(Modifier.height(32.dp))

        if (uiState.loading) {
            CircularProgressIndicator(color = NeonCyan)
        } else {
            Button(
                onClick = { viewModel.submitComplete(onSetupComplete) },
                colors = ButtonDefaults.buttonColors(
                    containerColor = NeonCyan,
                    contentColor = DeepBlack,
                ),
                modifier = Modifier.fillMaxWidth(0.6f),
            ) {
                Text("Done")
            }
        }
    }
}
