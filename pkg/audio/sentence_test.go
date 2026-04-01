package audio

import (
	"reflect"
	"testing"
)

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "empty input",
			in:   "",
			want: nil,
		},
		{
			name: "single sentence",
			in:   "Hello world.",
			want: []string{"Hello world."},
		},
		{
			name: "decimal numbers do not split",
			in:   "The value is 3.14 today. Keep watching closely.",
			want: []string{"The value is 3.14 today.", "Keep watching closely."},
		},
		{
			name: "newline boundary",
			in:   "This is line number one\nThis is line number two",
			want: []string{"This is line number one", "This is line number two"},
		},
		{
			name: "newline with surrounding spaces",
			in:   "  This is the first line   \n   This is the second line   ",
			want: []string{"This is the first line", "This is the second line"},
		},
		{
			name: "trailing punctuation consumed",
			in:   "Please wait a moment... What on earth?! That is perfectly fine.",
			want: []string{"Please wait a moment...", "What on earth?!", "That is perfectly fine."},
		},
		{
			name: "short leading fragment merges with next",
			in:   "Hi. This is a longer sentence.",
			want: []string{"Hi. This is a longer sentence."},
		},
		{
			name: "consecutive short fragments keep merging",
			in:   "A. B. C. This is the real sentence.",
			want: []string{"A. B. C. This is the real sentence."},
		},
		{
			name: "short trailing fragment merges back",
			in:   "This sentence is long enough. End.",
			want: []string{"This sentence is long enough. End."},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitSentences(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SplitSentences(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}
