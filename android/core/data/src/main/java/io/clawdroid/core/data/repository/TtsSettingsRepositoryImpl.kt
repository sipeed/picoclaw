package io.clawdroid.core.data.repository

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.floatPreferencesKey
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import io.clawdroid.core.domain.model.TtsConfig
import io.clawdroid.core.domain.repository.TtsSettingsRepository
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

private val Context.ttsDataStore: DataStore<Preferences> by preferencesDataStore(name = "tts_settings")

class TtsSettingsRepositoryImpl(
    private val context: Context
) : TtsSettingsRepository {

    private object Keys {
        val ENGINE = stringPreferencesKey("engine_package")
        val VOICE_NAME = stringPreferencesKey("voice_name")
        val SPEECH_RATE = floatPreferencesKey("speech_rate")
        val PITCH = floatPreferencesKey("pitch")
    }

    override val ttsConfig: Flow<TtsConfig> = context.ttsDataStore.data.map { prefs ->
        TtsConfig(
            enginePackageName = prefs[Keys.ENGINE],
            voiceName = prefs[Keys.VOICE_NAME],
            speechRate = prefs[Keys.SPEECH_RATE] ?: 1.0f,
            pitch = prefs[Keys.PITCH] ?: 1.0f
        )
    }

    override suspend fun updateEngine(packageName: String?) {
        context.ttsDataStore.edit { prefs ->
            if (packageName != null) prefs[Keys.ENGINE] = packageName
            else prefs.remove(Keys.ENGINE)
            // エンジン変更時は音声選択をリセット
            prefs.remove(Keys.VOICE_NAME)
        }
    }

    override suspend fun updateVoiceName(voiceName: String?) {
        context.ttsDataStore.edit { prefs ->
            if (voiceName != null) prefs[Keys.VOICE_NAME] = voiceName
            else prefs.remove(Keys.VOICE_NAME)
        }
    }

    override suspend fun updateSpeechRate(rate: Float) {
        context.ttsDataStore.edit { it[Keys.SPEECH_RATE] = rate.coerceIn(0.5f, 2.0f) }
    }

    override suspend fun updatePitch(pitch: Float) {
        context.ttsDataStore.edit { it[Keys.PITCH] = pitch.coerceIn(0.5f, 2.0f) }
    }
}
