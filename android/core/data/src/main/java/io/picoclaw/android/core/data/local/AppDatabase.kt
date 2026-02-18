package io.picoclaw.android.core.data.local

import androidx.room.Database
import androidx.room.RoomDatabase
import androidx.room.TypeConverters
import io.picoclaw.android.core.data.local.converter.Converters
import io.picoclaw.android.core.data.local.dao.MessageDao
import io.picoclaw.android.core.data.local.entity.MessageEntity

@Database(entities = [MessageEntity::class], version = 1, exportSchema = false)
@TypeConverters(Converters::class)
abstract class AppDatabase : RoomDatabase() {
    abstract fun messageDao(): MessageDao
}
