package io.clawdroid.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import io.clawdroid.core.data.local.dao.MessageDao
import io.clawdroid.core.data.mapper.MessageMapper
import io.clawdroid.core.data.remote.dto.WsOutgoing
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

class AgentMessageReceiver : BroadcastReceiver(), KoinComponent {

    private val messageDao: MessageDao by inject()
    private val scope: CoroutineScope by inject()
    private val json = Json { ignoreUnknownKeys = true }

    override fun onReceive(context: Context, intent: Intent) {
        val messageJson = intent.getStringExtra("message") ?: return

        Log.d(TAG, "Received broadcast message: ${messageJson.take(100)}")

        val pendingResult = goAsync()

        try {
            val msg = json.decodeFromString<WsOutgoing>(messageJson)

            // Skip ephemeral status messages
            if (msg.type == "status" || msg.type == "status_end") {
                pendingResult.finish()
                return
            }

            // Save to DB then finish
            val entity = MessageMapper.toEntity(msg)
            scope.launch {
                try {
                    messageDao.insert(entity)
                } catch (e: Exception) {
                    Log.e(TAG, "Failed to insert message to DB", e)
                } finally {
                    pendingResult.finish()
                }
            }

            // Show notification (synchronous, safe to call here)
            NotificationHelper.showMessageNotification(context, msg.content)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to process broadcast message", e)
            pendingResult.finish()
        }
    }

    companion object {
        private const val TAG = "AgentMessageReceiver"
        const val ACTION = "io.clawdroid.AGENT_MESSAGE"
    }
}
