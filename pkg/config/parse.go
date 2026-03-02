package config

func GetMap(data map[string]any, key string) (map[string]any, bool) {
	v, ok := data[key]
	if !ok {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

func GetString(data map[string]any, key string) (string, bool) {
	v, ok := data[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func GetStringOrDefault(data map[string]any, key string, defaultVal string) string {
	if v, ok := GetString(data, key); ok {
		return v
	}
	return defaultVal
}

func GetFloat(data map[string]any, key string) (float64, bool) {
	v, ok := data[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

func GetBool(data map[string]any, key string) (bool, bool) {
	v, ok := data[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func GetBoolOrDefault(data map[string]any, key string, defaultVal bool) bool {
	if v, ok := GetBool(data, key); ok {
		return v
	}
	return defaultVal
}

func GetStringSlice(data map[string]any, key string) []string {
	v, ok := data[key]
	if !ok {
		return []string{}
	}
	arr, ok := v.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func GetInt(m map[string]any, key string) (int, bool) {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n), true
		case int:
			return n, true
		}
	}
	return 0, false
}

func GetIntOrDefault(data map[string]any, key string, defaultVal int) int {
	if v, ok := GetInt(data, key); ok {
		return v
	}
	return defaultVal
}
