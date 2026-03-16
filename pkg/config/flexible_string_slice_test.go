package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFlexibleStringSliceUnmarshalJSON_String(t *testing.T) {
	var got FlexibleStringSlice
	if err := json.Unmarshal([]byte(`"general, #ops，dev"`), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := FlexibleStringSlice{"general", "#ops", "dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FlexibleStringSlice = %#v, want %#v", got, want)
	}
}
