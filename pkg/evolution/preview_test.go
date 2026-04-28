package evolution

import (
	"strings"
	"testing"
)

func TestBuildLineDiffPreview_UsesUnifiedDiffStyle(t *testing.T) {
	current := strings.Join([]string{
		"---",
		"name: weather",
		"description: weather helper",
		"---",
		"# Weather",
		"## Start Here",
		"Use city names first.",
		"",
	}, "\n")
	rendered := strings.Join([]string{
		"---",
		"name: weather",
		"description: weather helper",
		"---",
		"# Weather",
		"## Start Here",
		"Use city names first.",
		"",
		"## Start Here",
		"Use native-name query first.",
		"",
	}, "\n")

	diff := buildLineDiffPreview(current, rendered)

	for _, want := range []string{
		"--- current",
		"+++ rendered",
		"@@",
		"+## Start Here",
		"+Use native-name query first.",
	} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff missing %q:\n%s", want, diff)
		}
	}
}

func TestBuildLineDiffPreview_NoContentChange(t *testing.T) {
	body := "---\nname: weather\n---\n# Weather\n"
	diff := buildLineDiffPreview(body, body)
	if diff != "(no content change)" {
		t.Fatalf("diff = %q, want no-content marker", diff)
	}
}

func TestBuildLineDiffPreview_LimitsContextAroundChanges(t *testing.T) {
	current := strings.Join([]string{
		"line-01",
		"line-02",
		"line-03",
		"line-04",
		"line-05",
		"line-06",
		"line-07",
		"line-08",
		"line-09",
		"line-10",
		"",
	}, "\n")
	rendered := strings.Join([]string{
		"line-01",
		"line-02",
		"line-03",
		"line-04",
		"line-05",
		"line-06",
		"inserted-a",
		"inserted-b",
		"line-07",
		"line-08",
		"line-09",
		"line-10",
		"",
	}, "\n")

	diff := buildLineDiffPreview(current, rendered)

	for _, want := range []string{
		"@@",
		" line-05",
		" line-06",
		"+inserted-a",
		"+inserted-b",
		" line-07",
		" line-08",
	} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff missing %q:\n%s", want, diff)
		}
	}
	for _, unwanted := range []string{
		"line-01",
		"line-02",
		"line-09",
		"line-10",
	} {
		if strings.Contains(diff, unwanted) {
			t.Fatalf("diff should omit distant context %q:\n%s", unwanted, diff)
		}
	}
}
