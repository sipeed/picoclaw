plugins {
    alias(libs.plugins.android.library)
}

android {
    namespace = "io.picoclaw.android.core.domain"
    compileSdk = 35

    defaultConfig {
        minSdk = 28
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(libs.coroutines.android)
}
