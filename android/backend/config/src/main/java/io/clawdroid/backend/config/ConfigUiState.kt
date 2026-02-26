package io.clawdroid.backend.config

data class ConfigUiState(
    val listState: ListState = ListState.Loading,
    val detailState: DetailState? = null,
    val saveState: SaveState = SaveState.Idle,
)

sealed interface ListState {
    data object Loading : ListState
    data class Error(val message: String) : ListState
    data class Loaded(val sections: List<SectionSummary>) : ListState
}

data class DetailState(
    val sectionKey: String,
    val sectionLabel: String,
    val fields: List<FieldState>,
)

data class SectionSummary(val key: String, val label: String, val fieldCount: Int)

data class FieldState(
    val key: String,
    val label: String,
    val type: String,
    val secret: Boolean,
    val value: String,
    val originalValue: String,
)

sealed interface SaveState {
    data object Idle : SaveState
    data object Saving : SaveState
    data class Success(val restart: Boolean) : SaveState
    data class Error(val message: String) : SaveState
}
