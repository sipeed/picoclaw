package io.clawdroid.backend.loader

import android.content.Context
import android.util.Log
import io.clawdroid.backend.api.BackendState
import io.clawdroid.backend.api.GatewaySettingsStore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.io.BufferedReader
import java.net.HttpURLConnection
import java.net.URL
import java.util.UUID

class GatewayProcessManager(
    private val context: Context,
    private val settingsStore: GatewaySettingsStore,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val _state = MutableStateFlow(BackendState.STOPPED)
    val state: StateFlow<BackendState> = _state.asStateFlow()

    private var process: Process? = null
    private var logJob: Job? = null
    private var watchdogJob: Job? = null
    private var healthCheckJob: Job? = null
    private var settingsObserverJob: Job? = null

    private val binaryPath: String
        get() = context.applicationInfo.nativeLibraryDir + "/libclawdroid.so"

    suspend fun start() {
        ensureApiKey()
        startProcess()
        startSettingsObserver()
    }

    suspend fun stop() {
        settingsObserverJob?.cancel()
        settingsObserverJob = null
        stopProcess()
    }

    private suspend fun ensureApiKey() {
        val settings = settingsStore.settings.value
        if (settings.apiKey.isEmpty()) {
            settingsStore.update(settings.copy(apiKey = UUID.randomUUID().toString()))
        }
    }

    private fun startProcess() {
        _state.value = BackendState.STARTING

        val settings = settingsStore.settings.value
        val env = mapOf(
            "HOME" to context.filesDir.absolutePath,
            "CLAWDROID_GATEWAY_API_KEY" to settings.apiKey,
        )

        val pb = ProcessBuilder(binaryPath, "gateway", "run")
        pb.environment().putAll(env)
        pb.directory(context.filesDir)
        pb.redirectErrorStream(true)

        val proc = pb.start()
        process = proc
        Log.i(TAG, "Started gateway process")

        logJob = scope.launch { forwardLogs(proc) }
        watchdogJob = scope.launch { watchProcess(proc) }
        healthCheckJob = scope.launch { pollHealth(settings.httpPort) }
    }

    private fun stopProcess() {
        healthCheckJob?.cancel()
        healthCheckJob = null
        watchdogJob?.cancel()
        watchdogJob = null
        logJob?.cancel()
        logJob = null

        process?.let { proc ->
            proc.destroy()
            proc.waitFor()
            Log.i(TAG, "Stopped gateway process")
        }
        process = null
        _state.value = BackendState.STOPPED
    }

    private fun forwardLogs(proc: Process) {
        proc.inputStream.bufferedReader().use { reader: BufferedReader ->
            reader.forEachLine { line ->
                Log.i(TAG, line)
            }
        }
    }

    private suspend fun pollHealth(port: Int) {
        val url = "http://127.0.0.1:$port/api/config"
        while (scope.isActive) {
            try {
                val conn = URL(url).openConnection() as HttpURLConnection
                conn.connectTimeout = 1000
                conn.readTimeout = 1000
                try {
                    if (conn.responseCode == 200) {
                        if (_state.value == BackendState.STARTING) {
                            _state.value = BackendState.RUNNING
                            Log.i(TAG, "Gateway health check passed")
                        }
                    }
                } finally {
                    conn.disconnect()
                }
            } catch (_: Exception) {
                // Not ready yet
            }
            delay(500)
        }
    }

    private suspend fun watchProcess(proc: Process) {
        var backoffMs = 1000L
        proc.waitFor()
        val exitCode = proc.exitValue()
        Log.w(TAG, "Gateway process exited: code=$exitCode")

        if (_state.value == BackendState.STOPPED) return

        _state.value = BackendState.ERROR

        while (scope.isActive && _state.value == BackendState.ERROR) {
            Log.i(TAG, "Restarting gateway process (backoff=${backoffMs}ms)")
            delay(backoffMs)
            backoffMs = (backoffMs * 2).coerceAtMost(30_000)
            try {
                stopProcess()
                startProcess()
                return
            } catch (e: Exception) {
                Log.e(TAG, "Restart failed", e)
                _state.value = BackendState.ERROR
            }
        }
    }

    private fun startSettingsObserver() {
        settingsObserverJob = scope.launch {
            settingsStore.settings
                .map { it.apiKey }
                .distinctUntilChanged()
                .drop(1)
                .collect { apiKey ->
                    if (process != null && apiKey.isNotEmpty()) {
                        Log.i(TAG, "API key changed, restarting gateway")
                        stopProcess()
                        startProcess()
                    }
                }
        }
    }

    companion object {
        private const val TAG = "GatewayProcess"
    }
}
