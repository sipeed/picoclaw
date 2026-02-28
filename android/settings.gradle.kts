pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}
dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}
rootProject.name = "ClawDroid"
include(":app")
include(":feature:chat")
include(":core:domain")
include(":core:data")
include(":core:ui")
include(":backend:api")
include(":backend:config")
include(":backend:loader-noop")
