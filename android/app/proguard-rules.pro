# ========================
# kotlinx.serialization
# ========================
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt

-keepclassmembers class kotlinx.serialization.json.** { *** Companion; }

# Keep @Serializable classes and their generated serializers
-keepclassmembers @kotlinx.serialization.Serializable class ** {
    *** Companion;
    *** INSTANCE;
    kotlinx.serialization.KSerializer serializer(...);
}
-keepclasseswithmembers class io.clawdroid.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# Keep generated $$serializer classes
-keepclassmembers class io.clawdroid.**$$serializer {
    *** INSTANCE;
    *;
}

# Keep private ImageEntry used via Json.decodeFromString in MessageMapper
-keep class io.clawdroid.core.data.mapper.ImageEntry { *; }

# ========================
# Enums stored as strings in Room DB
# ========================
-keepclassmembers enum io.clawdroid.core.domain.model.MessageSender { *; }
-keepclassmembers enum io.clawdroid.core.domain.model.MessageStatus { *; }

# ========================
# Ktor
# ========================
-keep class io.ktor.** { *; }
-keepclassmembers class io.ktor.** { volatile <fields>; }
-dontwarn io.ktor.**

# ========================
# OkHttp / Okio
# ========================
-dontwarn okhttp3.**
-dontwarn okio.**

# ========================
# Koin / ViewModels
# ========================
-keep class * extends androidx.lifecycle.ViewModel { <init>(...); }
