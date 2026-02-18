package io.picoclaw.android

import android.app.Application
import io.picoclaw.android.di.appModule
import org.koin.android.ext.koin.androidContext
import org.koin.core.context.startKoin

class PicoClawApp : Application() {
    override fun onCreate() {
        super.onCreate()
        startKoin {
            androidContext(this@PicoClawApp)
            modules(appModule)
        }
    }
}
