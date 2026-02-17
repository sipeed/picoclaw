package main

import (
	"reflect"
	"testing"
)

func TestParseGlobalFlags_ConfigPair(t *testing.T) {
	args := []string{"picoclaw", "--config", "/tmp/config.json", "gateway", "--debug"}

	filtered, override, err := parseGlobalFlags(args)
	if err != nil {
		t.Fatalf("parseGlobalFlags() error: %v", err)
	}
	if override != "/tmp/config.json" {
		t.Errorf("override = %q, want %q", override, "/tmp/config.json")
	}

	want := []string{"picoclaw", "gateway", "--debug"}
	if !reflect.DeepEqual(filtered, want) {
		t.Errorf("filtered args = %#v, want %#v", filtered, want)
	}
}

func TestParseGlobalFlags_ConfigEqualsSyntax(t *testing.T) {
	args := []string{"picoclaw", "--config=/tmp/config.json", "status"}

	filtered, override, err := parseGlobalFlags(args)
	if err != nil {
		t.Fatalf("parseGlobalFlags() error: %v", err)
	}
	if override != "/tmp/config.json" {
		t.Errorf("override = %q, want %q", override, "/tmp/config.json")
	}

	want := []string{"picoclaw", "status"}
	if !reflect.DeepEqual(filtered, want) {
		t.Errorf("filtered args = %#v, want %#v", filtered, want)
	}
}

func TestParseGlobalFlags_MissingValue(t *testing.T) {
	tests := [][]string{
		{"picoclaw", "--config"},
		{"picoclaw", "--config", ""},
		{"picoclaw", "--config= "},
	}

	for _, tt := range tests {
		_, _, err := parseGlobalFlags(tt)
		if err == nil {
			t.Errorf("parseGlobalFlags(%#v) expected error, got nil", tt)
		}
	}
}

func TestParseGlobalFlags_DoesNotConsumeCronShortFlag(t *testing.T) {
	args := []string{"picoclaw", "cron", "add", "-c", "* * * * *"}

	filtered, override, err := parseGlobalFlags(args)
	if err != nil {
		t.Fatalf("parseGlobalFlags() error: %v", err)
	}
	if override != "" {
		t.Errorf("override = %q, want empty", override)
	}
	if !reflect.DeepEqual(filtered, args) {
		t.Errorf("filtered args = %#v, want %#v", filtered, args)
	}
}
