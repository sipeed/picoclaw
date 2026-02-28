package io.clawdroid.di

import io.clawdroid.backend.api.BackendLifecycle
import io.clawdroid.backend.api.NoopBackendLifecycle
import org.koin.dsl.module

val flavorModule = module {
    // Step 13 で EmbeddedBackendLifecycle に差し替え予定
    single<BackendLifecycle> { NoopBackendLifecycle() }
}
