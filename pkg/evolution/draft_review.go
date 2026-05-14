package evolution

import "strings"

type DraftReviewResult struct {
	Status      DraftStatus
	Findings    []string
	ReviewNotes []string
}

func ReviewDraft(draft SkillDraft) DraftReviewResult {
	findings := append([]string(nil), ValidateDraft(draft)...)
	findings = append(findings, scanDraftContent(draft)...)

	result := DraftReviewResult{
		Status:      DraftStatusCandidate,
		Findings:    findings,
		ReviewNotes: []string{"local structural validation completed"},
	}
	if len(findings) > 0 {
		result.Status = DraftStatusQuarantined
	}
	return result
}

func scanDraftContent(draft SkillDraft) []string {
	body := strings.ToLower(draft.BodyOrPatch)
	findings := make([]string, 0, 2)

	if strings.Contains(body, "sk-live-") || strings.Contains(body, "sk_test_") || strings.Contains(body, "api_key=") {
		findings = append(findings, "secret-like token detected in body_or_patch")
	}
	if strings.Contains(body, "-----begin private key-----") {
		findings = append(findings, "private key material detected in body_or_patch")
	}

	return findings
}
