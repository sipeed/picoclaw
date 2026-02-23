package io.clawdroid

import android.app.Application
import io.clawdroid.di.appModule
import io.clawdroid.receiver.NotificationHelper
import org.koin.android.ext.koin.androidContext
import org.koin.core.context.startKoin

class ClawDroidApp : Application() {
    override fun onCreate() {
        super.onCreate()
        startKoin {
            androidContext(this@ClawDroidApp)
            modules(appModule)
        }
        NotificationHelper.createNotificationChannel(this)
    }
}
