package evolution

func isTaskRecordKind(kind RecordKind) bool {
	return kind == RecordKindTask || kind == legacyRecordKindCase
}

func isPatternRecordKind(kind RecordKind) bool {
	return kind == RecordKindPattern || kind == legacyRecordKindRule
}
