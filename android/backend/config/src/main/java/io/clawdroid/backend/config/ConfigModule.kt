package io.clawdroid.backend.config

import org.koin.core.definition.Callbacks
import org.koin.core.module.dsl.viewModel
import org.koin.core.module.dsl.withOptions
import org.koin.dsl.module

val configModule = module {
    single { ConfigApiClient() } withOptions { callbacks = Callbacks(onClose = { it?.close() }) }
    viewModel { ConfigViewModel(get()) }
}
