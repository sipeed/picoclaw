package io.clawdroid.settings

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.intPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import io.clawdroid.backend.api.GatewaySettings
import io.clawdroid.backend.api.GatewaySettingsStore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn

private val Context.gatewayDataStore by preferencesDataStore(name = "gateway_settings")

class GatewaySettingsStoreImpl(
    private val context: Context,
    scope: CoroutineScope,
) : GatewaySettingsStore {

    private object Keys {
        val HTTP_PORT = intPreferencesKey("http_port")
        val API_KEY = stringPreferencesKey("api_key")
    }

    override val settings: StateFlow<GatewaySettings> =
        context.gatewayDataStore.data.map { prefs ->
            GatewaySettings(
                httpPort = prefs[Keys.HTTP_PORT] ?: DEFAULT.httpPort,
                apiKey = prefs[Keys.API_KEY] ?: DEFAULT.apiKey,
            )
        }.stateIn(scope, SharingStarted.Eagerly, DEFAULT)

    companion object {
        private val DEFAULT = GatewaySettings()
    }

    override suspend fun update(settings: GatewaySettings) {
        context.gatewayDataStore.edit { prefs ->
            prefs[Keys.HTTP_PORT] = settings.httpPort
            prefs[Keys.API_KEY] = settings.apiKey
        }
    }
}
