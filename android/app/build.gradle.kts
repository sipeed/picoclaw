import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.compose.compiler)
    alias(libs.plugins.serialization)
}

val keystorePropertiesFile = rootProject.file("keystore.properties")
val keystoreProperties = Properties().apply {
    if (keystorePropertiesFile.exists()) {
        keystorePropertiesFile.inputStream().use { load(it) }
    }
}

val versionFile = rootProject.file("../VERSION")
val appVersionName = if (versionFile.exists()) versionFile.readText().trim() else "0.1.0"

android {
    namespace = "io.clawdroid"
    compileSdk = libs.versions.android.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "io.clawdroid"
        minSdk = libs.versions.android.minSdk.get().toInt()
        targetSdk = libs.versions.android.targetSdk.get().toInt()
        versionCode = 1
        versionName = appVersionName
    }

    signingConfigs {
        create("release") {
            storeFile = keystoreProperties.getProperty("storeFile", "")
                .takeIf { it.isNotEmpty() }?.let { rootProject.file(it) }
            storePassword = keystoreProperties.getProperty("storePassword", "")
            keyAlias = keystoreProperties.getProperty("keyAlias", "")
            keyPassword = keystoreProperties.getProperty("keyPassword", "")
        }
    }

    buildTypes {
        debug {
            signingConfig = signingConfigs.getByName("release")
        }
        release {
            isMinifyEnabled = true
            signingConfig = signingConfigs.getByName("release")
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }

    flavorDimensions += "variant"
    productFlavors {
        create("termux") { dimension = "variant" }
        create("embedded") { dimension = "variant" }
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    packaging {
        jniLibs.useLegacyPackaging = true
    }

    splits {
        abi {
            isEnable = project.hasProperty("enableAbiSplit")
            reset()
            include("arm64-v8a", "armeabi-v7a", "x86_64")
            isUniversalApk = true
        }
    }

    sourceSets {
        getByName("termux") { java.srcDirs("src/termux/java") }
        getByName("embedded") { java.srcDirs("src/embedded/java") }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(project(":feature:chat"))
    implementation(project(":core:domain"))
    implementation(project(":core:data"))
    implementation(project(":core:ui"))
    implementation(project(":backend:api"))
    implementation(project(":backend:config"))

    "termuxImplementation"(project(":backend:loader-noop"))
    "embeddedImplementation"(project(":backend:loader"))

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.material3)
    implementation(libs.activity.compose)
    implementation(libs.core.ktx)
    implementation(libs.lifecycle.runtime.compose)
    implementation(libs.lifecycle.service)
    implementation(libs.navigation.compose)

    implementation(libs.koin.android)
    implementation(libs.koin.compose)

    implementation(libs.ktor.client.okhttp)
    implementation(libs.ktor.client.websockets)

    implementation(libs.room.runtime)
    implementation(libs.room.ktx)

    implementation(libs.coroutines.android)

    implementation(libs.datastore.preferences)

    implementation(libs.serialization.json)

    implementation(libs.icons.lucide)

    debugImplementation(libs.compose.ui.tooling)
}
