package io.clawdroid.di

import io.clawdroid.backend.api.BackendLifecycle
import io.clawdroid.backend.loader.EmbeddedBackendLifecycle
import io.clawdroid.backend.loader.GatewayProcessManager
import org.koin.android.ext.koin.androidContext
import org.koin.dsl.module

val flavorModule = module {
    single { GatewayProcessManager(androidContext(), get()) }
    single<BackendLifecycle> { EmbeddedBackendLifecycle(androidContext(), get()) }
}
