plugins {
    alias(libs.plugins.android.library)
}

android {
    namespace = "io.clawdroid.backend.loader"
    compileSdk = libs.versions.android.compileSdk.get().toInt()
    defaultConfig { minSdk = libs.versions.android.minSdk.get().toInt() }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(project(":backend:api"))
    implementation(libs.coroutines.android)
}
