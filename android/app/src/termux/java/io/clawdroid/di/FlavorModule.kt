package io.clawdroid.di

import io.clawdroid.backend.api.BackendLifecycle
import io.clawdroid.backend.api.NoopBackendLifecycle
import org.koin.dsl.module

val flavorModule = module {
    single<BackendLifecycle> { NoopBackendLifecycle() }
}
