package io.clawdroid.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.clawdroid.backend.api.GatewaySettings
import io.clawdroid.backend.api.GatewaySettingsStore
import io.clawdroid.backend.config.ConfigApiClient
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

data class AppSettingsUiState(
    val apiKey: String = "",
    val httpPort: String = "18790",
    val saving: Boolean = false,
    val error: String? = null,
) {
    val httpPortError: String? get() = portError(httpPort)
    val hasErrors: Boolean get() = httpPortError != null
}

private fun portError(value: String): String? {
    if (value.isEmpty()) return null
    val port = value.toIntOrNull() ?: return "Invalid number"
    return if (port !in 1..65535) "1-65535" else null
}

class AppSettingsViewModel(
    private val settingsStore: GatewaySettingsStore,
    private val configApiClient: ConfigApiClient,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AppSettingsUiState())
    val uiState: StateFlow<AppSettingsUiState> = _uiState.asStateFlow()

    init {
        val current = settingsStore.settings.value
        _uiState.value = AppSettingsUiState(
            apiKey = current.apiKey,
            httpPort = current.httpPort.toString(),
        )
    }

    fun onApiKeyChange(value: String) {
        _uiState.update { it.copy(apiKey = value, error = null) }
    }

    fun onHttpPortChange(value: String) {
        if (value.isEmpty() || value.toIntOrNull() != null) {
            _uiState.update { it.copy(httpPort = value, error = null) }
        }
    }

    fun save(onComplete: () -> Unit) {
        viewModelScope.launch {
            val state = _uiState.value
            if (state.hasErrors || state.saving) return@launch

            _uiState.update { it.copy(saving = true, error = null) }

            val defaults = GatewaySettings()
            val newPort = state.httpPort.toIntOrNull()?.takeIf { it in 1..65535 } ?: defaults.httpPort
            val newKey = state.apiKey

            // Send update via config API using current (old) connection settings
            val payload = buildJsonObject {
                put("gateway", buildJsonObject {
                    put("port", JsonPrimitive(newPort))
                    put("api_key", JsonPrimitive(newKey))
                })
            }

            try {
                configApiClient.saveConfig(payload)
                // Persist new values locally after remote success
                settingsStore.update(GatewaySettings(httpPort = newPort, apiKey = newKey))
                _uiState.update { it.copy(saving = false) }
                onComplete()
            } catch (e: Exception) {
                _uiState.update { it.copy(saving = false, error = e.message ?: "Save failed") }
            }
        }
    }
}
