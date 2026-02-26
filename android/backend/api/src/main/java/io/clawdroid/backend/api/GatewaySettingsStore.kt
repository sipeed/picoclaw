package io.clawdroid.backend.api

import kotlinx.coroutines.flow.StateFlow

interface GatewaySettingsStore {
    val settings: StateFlow<GatewaySettings>
    suspend fun update(settings: GatewaySettings)
}
