package io.clawdroid.backend.api

import kotlinx.coroutines.flow.StateFlow

interface BackendLifecycle {
    val state: StateFlow<BackendState>
    val isManaged: Boolean
    suspend fun start()
    suspend fun stop()
}
