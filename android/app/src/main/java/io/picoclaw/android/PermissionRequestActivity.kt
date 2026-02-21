package io.picoclaw.android

import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat

class PermissionRequestActivity : ComponentActivity() {

    private val launcher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { granted ->
        broadcastResult(
            intent.getStringExtra(EXTRA_PERMISSION).orEmpty(),
            granted
        )
        finish()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val permission = intent.getStringExtra(EXTRA_PERMISSION)
        if (permission == null) {
            finish()
            return
        }
        if (ContextCompat.checkSelfPermission(this, permission)
            == PackageManager.PERMISSION_GRANTED
        ) {
            broadcastResult(permission, true)
            finish()
            return
        }
        launcher.launch(permission)
    }

    private fun broadcastResult(permission: String, granted: Boolean) {
        sendBroadcast(
            Intent(ACTION_RESULT)
                .setPackage(packageName)
                .putExtra(EXTRA_PERMISSION, permission)
                .putExtra(EXTRA_GRANTED, granted)
        )
    }

    companion object {
        const val EXTRA_PERMISSION = "permission"
        const val EXTRA_GRANTED = "granted"
        const val ACTION_RESULT = "io.picoclaw.android.PERMISSION_RESULT"

        fun intent(context: Context, permission: String): Intent =
            Intent(context, PermissionRequestActivity::class.java)
                .putExtra(EXTRA_PERMISSION, permission)
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }
}
