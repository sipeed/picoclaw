package io.clawdroid.backend.config

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put

class ConfigViewModel(private val apiClient: ConfigApiClient) : ViewModel() {

    private val _uiState = MutableStateFlow(ConfigUiState())
    val uiState: StateFlow<ConfigUiState> = _uiState.asStateFlow()

    private var schema: ConfigSchema? = null
    private var configValues: JsonObject? = null
    private var saveJob: Job? = null
    private var pendingSectionKey: String? = null

    init {
        loadData()
    }

    fun retry() {
        loadData()
    }

    fun onSectionSelected(sectionKey: String) {
        val current = _uiState.value.detailState
        if (current != null && current.sectionKey == sectionKey) return
        if (schema == null) {
            pendingSectionKey = sectionKey
            return
        }
        loadSection(sectionKey)
    }

    private fun loadSection(sectionKey: String) {
        val s = schema ?: return
        val config = configValues ?: return
        val section = s.sections.find { it.key == sectionKey } ?: return

        val fields = section.fields.map { field ->
            val fullKey = "$sectionKey.${field.key}"
            val raw = extractNestedValue(config, fullKey)
            val display = jsonElementToEditString(raw, field.type)
            FieldState(
                key = field.key,
                label = field.label,
                type = field.type,
                secret = field.secret,
                value = display,
                originalValue = display,
            )
        }

        _uiState.update {
            it.copy(
                detailState = DetailState(
                    sectionKey = sectionKey,
                    sectionLabel = section.label,
                    fields = fields,
                ),
            )
        }
    }

    fun onFieldValueChanged(fieldKey: String, newValue: String) {
        _uiState.update { state ->
            val detail = state.detailState ?: return@update state
            val updated = detail.fields.map { field ->
                if (field.key == fieldKey) field.copy(value = newValue) else field
            }
            state.copy(detailState = detail.copy(fields = updated))
        }
    }

    fun onSave() {
        val detail = _uiState.value.detailState ?: return

        val changed = detail.fields.filter { it.value != it.originalValue }
        if (changed.isEmpty()) return

        _uiState.update { it.copy(saveState = SaveState.Saving) }

        saveJob = viewModelScope.launch {
            try {
                val prefixed = changed.map { it.copy(key = "${detail.sectionKey}.${it.key}") }
                val payload = buildNestedJsonObject(prefixed)
                val result = apiClient.saveConfig(payload)
                if (result.error != null) {
                    _uiState.update { it.copy(saveState = SaveState.Error(result.error)) }
                } else {
                    _uiState.update { it.copy(saveState = SaveState.Success(result.restart)) }
                    refreshConfig()
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(saveState = SaveState.Error(e.message ?: "Save failed"))
                }
            }
        }
    }

    fun onNavigateBackToList() {
        saveJob?.cancel()
        saveJob = null
        _uiState.update {
            it.copy(detailState = null, saveState = SaveState.Idle)
        }
    }

    fun dismissSaveResult() {
        _uiState.update { it.copy(saveState = SaveState.Idle) }
    }

    private fun loadData() {
        _uiState.update { it.copy(listState = ListState.Loading) }
        viewModelScope.launch {
            try {
                val schemaDeferred = async { apiClient.getSchema() }
                val configDeferred = async { apiClient.getConfig() }
                val s = schemaDeferred.await()
                val c = configDeferred.await()
                schema = s
                configValues = c
                _uiState.update {
                    it.copy(listState = ListState.Loaded(s.toSummaries()))
                }
                pendingSectionKey?.let { key ->
                    pendingSectionKey = null
                    loadSection(key)
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        listState = ListState.Error(
                            e.message ?: "Failed to load config"
                        )
                    )
                }
            }
        }
    }

    private suspend fun refreshConfig() {
        try {
            val c = apiClient.getConfig()
            configValues = c
            val current = _uiState.value.detailState
            if (current != null) {
                loadSection(current.sectionKey)
            }
        } catch (_: Exception) {
            // Refresh failure is non-critical; keep existing values
        }
    }

    private fun ConfigSchema.toSummaries(): List<SectionSummary> =
        sections.map { SectionSummary(it.key, it.label, it.fields.size) }

    // ── JSON mapping utilities ───────────────────────────────────

    private fun extractNestedValue(element: JsonElement, keyPath: String): JsonElement {
        val parts = keyPath.split(".")
        var current: JsonElement = element
        for (part in parts) {
            current = (current as? JsonObject)?.get(part) ?: return JsonNull
        }
        return current
    }

    private fun jsonElementToEditString(element: JsonElement, type: String): String {
        if (element is JsonNull) return ""
        return when (type) {
            "bool" -> element.jsonPrimitive.booleanOrNull?.toString() ?: ""
            "[]string" -> {
                if (element is JsonArray) {
                    element.map { it.jsonPrimitive.contentOrNull ?: "" }.joinToString(", ")
                } else ""
            }
            else -> (element as? JsonPrimitive)?.contentOrNull ?: element.toString()
        }
    }

    private fun buildNestedJsonObject(fields: List<FieldState>): JsonObject {
        val root = mutableMapOf<String, Any>()
        for (field in fields) {
            val parts = field.key.split(".")
            var current = root
            for (i in 0 until parts.size - 1) {
                @Suppress("UNCHECKED_CAST")
                current = current.getOrPut(parts[i]) { mutableMapOf<String, Any>() }
                    as MutableMap<String, Any>
            }
            current[parts.last()] = fieldValueToJsonElement(field)
        }
        return mapToJsonObject(root)
    }

    @Suppress("UNCHECKED_CAST")
    private fun mapToJsonObject(map: Map<String, Any>): JsonObject = buildJsonObject {
        for ((key, value) in map) {
            when (value) {
                is JsonElement -> put(key, value)
                is Map<*, *> -> put(key, mapToJsonObject(value as Map<String, Any>))
            }
        }
    }

    private fun fieldValueToJsonElement(field: FieldState): JsonElement = when (field.type) {
        "bool" -> JsonPrimitive(field.value.toBooleanStrictOrNull() ?: false)
        "int" -> JsonPrimitive(field.value.toLongOrNull() ?: 0L)
        "float" -> JsonPrimitive(field.value.toDoubleOrNull() ?: 0.0)
        "[]string" -> JsonArray(
            field.value.split(",")
                .map { it.trim() }
                .filter { it.isNotEmpty() }
                .map { JsonPrimitive(it) }
        )
        else -> JsonPrimitive(field.value)
    }
}
