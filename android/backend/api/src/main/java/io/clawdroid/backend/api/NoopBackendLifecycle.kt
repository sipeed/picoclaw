package io.clawdroid.backend.api

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

class NoopBackendLifecycle : BackendLifecycle {
    override val state: StateFlow<BackendState> = MutableStateFlow(BackendState.RUNNING)
    override val isManaged: Boolean = false
    override suspend fun start() {}
    override suspend fun stop() {}
}
