package io.clawdroid.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.clawdroid.backend.api.GatewaySettings
import io.clawdroid.backend.api.GatewaySettingsStore
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class AppSettingsUiState(
    val apiKey: String = "",
    val wsPort: String = "18793",
    val httpPort: String = "18790",
) {
    val wsPortError: String? get() = portError(wsPort)
    val httpPortError: String? get() = portError(httpPort)
    val hasErrors: Boolean get() = wsPortError != null || httpPortError != null
}

private fun portError(value: String): String? {
    if (value.isEmpty()) return null
    val port = value.toIntOrNull() ?: return "Invalid number"
    return if (port !in 1..65535) "1-65535" else null
}

class AppSettingsViewModel(
    private val settingsStore: GatewaySettingsStore,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AppSettingsUiState())
    val uiState: StateFlow<AppSettingsUiState> = _uiState.asStateFlow()

    init {
        val current = settingsStore.settings.value
        _uiState.value = AppSettingsUiState(
            apiKey = current.apiKey,
            wsPort = current.wsPort.toString(),
            httpPort = current.httpPort.toString(),
        )
    }

    fun onApiKeyChange(value: String) {
        _uiState.update { it.copy(apiKey = value) }
    }

    fun onWsPortChange(value: String) {
        if (value.isEmpty() || value.toIntOrNull() != null) {
            _uiState.update { it.copy(wsPort = value) }
        }
    }

    fun onHttpPortChange(value: String) {
        if (value.isEmpty() || value.toIntOrNull() != null) {
            _uiState.update { it.copy(httpPort = value) }
        }
    }

    fun save(onComplete: () -> Unit) {
        viewModelScope.launch {
            val state = _uiState.value
            if (state.hasErrors) return@launch
            val defaults = GatewaySettings()
            fun validPort(raw: String, fallback: Int): Int {
                val port = raw.toIntOrNull() ?: return fallback
                return if (port in 1..65535) port else fallback
            }
            val settings = GatewaySettings(
                wsPort = validPort(state.wsPort, defaults.wsPort),
                httpPort = validPort(state.httpPort, defaults.httpPort),
                apiKey = state.apiKey,
            )
            settingsStore.update(settings)
            onComplete()
        }
    }
}
