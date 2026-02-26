package io.clawdroid.backend.config

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.ui.res.painterResource
import com.composables.icons.lucide.R as LucideR
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.GlassBorder
import io.clawdroid.core.ui.theme.GlassWhite
import io.clawdroid.core.ui.theme.NeonCyan
import io.clawdroid.core.ui.theme.TextPrimary
import io.clawdroid.core.ui.theme.TextSecondary
import org.koin.androidx.compose.koinViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConfigSectionDetailScreen(
    sectionKey: String,
    onNavigateBack: () -> Unit,
    viewModel: ConfigViewModel = koinViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(sectionKey) {
        viewModel.onSectionSelected(sectionKey)
    }

    val detail = uiState.detailState
    val isDirty = detail?.fields?.any { it.value != it.originalValue } == true
    val isSaving = uiState.saveState is SaveState.Saving

    LaunchedEffect(uiState.saveState) {
        when (val state = uiState.saveState) {
            is SaveState.Success -> {
                val msg = if (state.restart) "Saved. Server restarting..." else "Saved."
                snackbarHostState.showSnackbar(msg)
                viewModel.dismissSaveResult()
            }
            is SaveState.Error -> {
                snackbarHostState.showSnackbar("Error: ${state.message}")
                viewModel.dismissSaveResult()
            }
            else -> {}
        }
    }

    ConfigBackground {
        Scaffold(
            containerColor = Color.Transparent,
            snackbarHost = { SnackbarHost(snackbarHostState) },
            topBar = {
                TopAppBar(
                    title = { Text(detail?.sectionLabel ?: "") },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = Color.Transparent,
                    ),
                    navigationIcon = {
                        IconButton(onClick = {
                            viewModel.onNavigateBackToList()
                            onNavigateBack()
                        }) {
                            Icon(
                                painter = painterResource(LucideR.drawable.lucide_ic_arrow_left),
                                contentDescription = "Back",
                                tint = TextSecondary,
                            )
                        }
                    },
                    actions = {
                        Button(
                            onClick = viewModel::onSave,
                            enabled = isDirty && !isSaving,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = NeonCyan,
                                contentColor = DeepBlack,
                                disabledContainerColor = NeonCyan.copy(alpha = 0.3f),
                                disabledContentColor = DeepBlack.copy(alpha = 0.5f),
                            ),
                            modifier = Modifier.padding(end = 8.dp),
                        ) {
                            if (isSaving) {
                                CircularProgressIndicator(
                                    color = DeepBlack,
                                    modifier = Modifier.size(18.dp),
                                    strokeWidth = 2.dp,
                                )
                            } else {
                                Text("Save")
                            }
                        }
                    },
                )
            },
        ) { padding ->
            if (detail != null) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding)
                        .padding(horizontal = 16.dp)
                        .verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    detail.fields.forEach { field ->
                        ConfigField(
                            field = field,
                            onValueChanged = { viewModel.onFieldValueChanged(field.key, it) },
                        )
                    }
                }
            } else {
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentAlignment = Alignment.Center,
                ) {
                    CircularProgressIndicator(color = NeonCyan)
                }
            }
        }
    }
}

@Composable
private fun ConfigField(
    field: FieldState,
    onValueChanged: (String) -> Unit,
) {
    when (field.type) {
        "bool" -> BoolField(field, onValueChanged)
        "int" -> NumberField(field, onValueChanged, KeyboardType.Number)
        "float" -> NumberField(field, onValueChanged, KeyboardType.Decimal)
        "[]string" -> StringArrayField(field, onValueChanged)
        "map", "[]any" -> ReadOnlyField(field)
        else -> StringField(field, onValueChanged)
    }
}

@Composable
private fun StringField(field: FieldState, onValueChanged: (String) -> Unit) {
    var hidden by remember(field.key) { mutableStateOf(field.secret) }

    OutlinedTextField(
        value = field.value,
        onValueChange = onValueChanged,
        label = { Text(field.label, color = TextSecondary) },
        singleLine = true,
        visualTransformation = if (hidden) PasswordVisualTransformation() else VisualTransformation.None,
        trailingIcon = if (field.secret) {
            {
                TextButton(onClick = { hidden = !hidden }) {
                    Text(
                        if (hidden) "Show" else "Hide",
                        color = NeonCyan,
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
            }
        } else null,
        colors = configFieldColors(),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun BoolField(field: FieldState, onValueChanged: (String) -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            field.label,
            style = MaterialTheme.typography.bodyLarge,
            color = TextPrimary,
        )
        Switch(
            checked = field.value.toBooleanStrictOrNull() == true,
            onCheckedChange = { onValueChanged(it.toString()) },
            colors = SwitchDefaults.colors(
                checkedThumbColor = NeonCyan,
                checkedTrackColor = NeonCyan.copy(alpha = 0.3f),
                uncheckedThumbColor = GlassWhite,
                uncheckedTrackColor = GlassBorder,
            ),
        )
    }
}

@Composable
private fun NumberField(
    field: FieldState,
    onValueChanged: (String) -> Unit,
    keyboardType: KeyboardType,
) {
    val filter: (String) -> Boolean = if (keyboardType == KeyboardType.Decimal) {
        { it.isEmpty() || it == "-" || it.toDoubleOrNull() != null }
    } else {
        { it.isEmpty() || it == "-" || it.toLongOrNull() != null }
    }

    OutlinedTextField(
        value = field.value,
        onValueChange = { if (filter(it)) onValueChanged(it) },
        label = { Text(field.label, color = TextSecondary) },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
        colors = configFieldColors(),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun StringArrayField(field: FieldState, onValueChanged: (String) -> Unit) {
    OutlinedTextField(
        value = field.value,
        onValueChange = onValueChanged,
        label = { Text(field.label, color = TextSecondary) },
        placeholder = { Text("Comma-separated values", color = TextSecondary.copy(alpha = 0.5f)) },
        singleLine = true,
        colors = configFieldColors(),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun ReadOnlyField(field: FieldState) {
    OutlinedTextField(
        value = field.value.ifEmpty { "(complex value)" },
        onValueChange = {},
        label = { Text(field.label, color = TextSecondary) },
        readOnly = true,
        enabled = false,
        colors = configFieldColors(),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun configFieldColors() = OutlinedTextFieldDefaults.colors(
    focusedBorderColor = NeonCyan.copy(alpha = 0.5f),
    unfocusedBorderColor = GlassBorder,
    focusedContainerColor = GlassWhite,
    unfocusedContainerColor = Color.Transparent,
    focusedTextColor = TextPrimary,
    unfocusedTextColor = TextPrimary,
)
