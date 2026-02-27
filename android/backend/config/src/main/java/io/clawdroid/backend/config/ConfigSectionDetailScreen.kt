package io.clawdroid.backend.config

import android.content.Intent
import android.net.Uri
import android.provider.DocumentsContract
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
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
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.platform.LocalContext
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
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConfigSectionDetailScreen(
    sectionKey: String,
    onNavigateBack: () -> Unit,
    viewModel: ConfigViewModel,
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
                    var lastGroup: String? = null
                    detail.fields.forEach { field ->
                        if (field.group != lastGroup) {
                            lastGroup = field.group
                            if (field.group.isNotEmpty()) {
                                Spacer(modifier = Modifier.height(8.dp))
                                HorizontalDivider(color = GlassBorder)
                                Text(
                                    text = field.group,
                                    style = MaterialTheme.typography.titleSmall,
                                    color = NeonCyan,
                                    modifier = Modifier.padding(top = 8.dp),
                                )
                            }
                        }
                        ConfigField(
                            field = field,
                            onValueChanged = { viewModel.onFieldValueChanged(field.key, it) },
                            snackbarHostState = snackbarHostState,
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
    snackbarHostState: SnackbarHostState? = null,
) {
    when (field.type) {
        "bool" -> BoolField(field, onValueChanged)
        "int" -> NumberField(field, onValueChanged, KeyboardType.Number)
        "float" -> NumberField(field, onValueChanged, KeyboardType.Decimal)
        "[]string" -> StringArrayField(field, onValueChanged)
        "directory" -> DirectoryField(field, onValueChanged, snackbarHostState)
        "map", "[]any" -> JsonField(field, onValueChanged)
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
private fun JsonField(field: FieldState, onValueChanged: (String) -> Unit) {
    val jsonError = remember(field.value) {
        if (field.value.isBlank()) null
        else try {
            kotlinx.serialization.json.Json.parseToJsonElement(field.value); null
        } catch (e: Exception) {
            e.message
        }
    }

    OutlinedTextField(
        value = field.value,
        onValueChange = onValueChanged,
        label = { Text(field.label, color = TextSecondary) },
        singleLine = false,
        minLines = 3,
        isError = jsonError != null,
        supportingText = if (jsonError != null) {
            { Text(jsonError, color = MaterialTheme.colorScheme.error) }
        } else null,
        colors = configFieldColors(),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun DirectoryField(
    field: FieldState,
    onValueChanged: (String) -> Unit,
    snackbarHostState: SnackbarHostState?,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()

    val launcher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.OpenDocumentTree()
    ) { uri: Uri? ->
        if (uri == null) return@rememberLauncherForActivityResult
        context.contentResolver.takePersistableUriPermission(
            uri,
            Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_GRANT_WRITE_URI_PERMISSION,
        )
        val path = safUriToPath(uri)
        if (path != null) {
            onValueChanged(path)
        } else {
            scope.launch {
                snackbarHostState?.showSnackbar("Internal storage only")
            }
        }
    }

    Column {
        OutlinedTextField(
            value = field.value,
            onValueChange = onValueChanged,
            label = { Text(field.label, color = TextSecondary) },
            singleLine = true,
            trailingIcon = {
                IconButton(onClick = { launcher.launch(null) }) {
                    Icon(
                        painter = painterResource(LucideR.drawable.lucide_ic_folder_open),
                        contentDescription = "Browse",
                        tint = NeonCyan,
                    )
                }
            },
            colors = configFieldColors(),
            modifier = Modifier.fillMaxWidth(),
        )
        Text(
            "Internal storage only for SAF picker",
            style = MaterialTheme.typography.labelSmall,
            color = TextSecondary.copy(alpha = 0.6f),
            modifier = Modifier.padding(start = 16.dp, top = 2.dp),
        )
    }
}

/**
 * Converts a SAF tree URI to a filesystem path.
 * Only internal storage (`primary:...`) is supported.
 */
private fun safUriToPath(uri: Uri): String? {
    val docId = try {
        DocumentsContract.getTreeDocumentId(uri)
    } catch (_: Exception) {
        return null
    }

    if (!docId.startsWith("primary:")) return null

    val relativePath = docId.removePrefix("primary:")
    return if (relativePath.isEmpty()) {
        "/storage/emulated/0"
    } else {
        "/storage/emulated/0/$relativePath"
    }
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
