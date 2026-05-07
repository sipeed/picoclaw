package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// diaIDRe matches valid dia_id patterns like "D1:3", "D30:5".
var diaIDRe = regexp.MustCompile(`^D(\d+):(\d+)$`)

// SplitEvidenceIDs splits an evidence string that may contain multiple
// semicolon-separated or space-separated dia_ids. Only returns valid IDs.
// Example: "D8:6; D9:17" → ["D8:6", "D9:17"]
// Example: "D9:1 D4:4 D4:6" → ["D9:1", "D4:4", "D4:6"]
func SplitEvidenceIDs(evidence string) []string {
	if evidence == "" {
		return nil
	}
	// Split on semicolons first, then spaces
	parts := strings.Split(evidence, ";")
	var ids []string
	for _, part := range parts {
		for _, token := range strings.Fields(strings.TrimSpace(part)) {
			token = strings.TrimSpace(token)
			if diaIDRe.MatchString(token) {
				ids = append(ids, NormalizeDiaID(token))
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

// NormalizeDiaID strips leading zeros from the number parts of a dia_id.
// "D30:05" → "D30:5", "D10:003" → "D10:3"
func NormalizeDiaID(id string) string {
	m := diaIDRe.FindStringSubmatch(id)
	if m == nil {
		return id
	}
	session, _ := strconv.Atoi(m[1])
	turn, _ := strconv.Atoi(m[2])
	return fmt.Sprintf("D%d:%d", session, turn)
}

// stopwords is a fixed English stopword list for deterministic keyword extraction.
var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {},
	"is": {}, "are": {}, "was": {}, "were": {},
	"did": {}, "does": {}, "do": {},
	"when": {}, "where": {}, "what": {}, "who": {},
	"how": {}, "why": {},
	"to": {}, "of": {}, "in": {}, "on": {}, "at": {},
	"for": {}, "and": {}, "or": {}, "but": {}, "not": {},
	"it": {}, "this": {}, "that": {}, "with": {},
	"from": {}, "by": {}, "as": {},
	"if": {}, "then": {}, "than": {}, "so": {},
	"no": {}, "yes": {},
	"all": {}, "any": {}, "each": {}, "every": {},
	"some": {}, "such": {},
	"about": {}, "into": {}, "over": {},
	"after": {}, "before": {}, "between": {},
	"through": {}, "during": {}, "until": {},
	"would": {}, "could": {}, "should": {},
	"may": {}, "might": {}, "can": {},
	"will": {}, "shall": {}, "must": {},
	"have": {}, "has": {}, "had": {},
	"been": {}, "being": {}, "be": {},
	"go": {}, "went": {}, "gone": {},
	"i": {}, "you": {}, "me": {}, "my": {}, "your": {},
	"we": {}, "they": {}, "them": {}, "our": {},
	"its": {}, "their": {}, "he": {}, "she": {},
	"his": {}, "her": {},
}

// ExtractKeywords removes stopwords and punctuation, returns individual keywords.
// Deterministic: uses fixed stopword list, no LLM.
func ExtractKeywords(question string) []string {
	// Lowercase and split on whitespace/punctuation
	lower := strings.ToLower(question)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var keywords []string
	for _, w := range words {
		if w == "" || len(w) < 2 {
			continue
		}
		if _, ok := stopwords[w]; ok {
			continue
		}
		keywords = append(keywords, w)
		if len(keywords) >= 6 {
			break
		}
	}
	return keywords
}

// TokenOverlapF1 computes token-level F1 between prediction and reference.
// Both strings are lowercased and split on whitespace.
// NOTE: This metric underestimates quality for multi-hop (cat 2) and
// open-ended (cat 3) questions where the gold answer uses different phrasing
// than the source text. LLM-Judge scoring is a v2 follow-up.
func TokenOverlapF1(prediction, reference string) float64 {
	predTokens := tokenize(prediction)
	refTokens := tokenize(reference)

	if len(predTokens) == 0 && len(refTokens) == 0 {
		return 1.0
	}
	if len(predTokens) == 0 || len(refTokens) == 0 {
		return 0.0
	}

	// Count matches
	refCount := map[string]int{}
	for _, t := range refTokens {
		refCount[t]++
	}

	predCount := map[string]int{}
	for _, t := range predTokens {
		predCount[t]++
	}

	var matches float64
	for token, pc := range predCount {
		if rc, ok := refCount[token]; ok {
			matches += float64(min(pc, rc))
		}
	}

	precision := matches / float64(len(predTokens))
	recall := matches / float64(len(refTokens))

	if precision+recall == 0 {
		return 0.0
	}
	return 2 * precision * recall / (precision + recall)
}

func tokenize(s string) []string {
	lower := strings.ToLower(s)
	return strings.Fields(lower)
}

// RecallHitRate computes fraction of evidence IDs found in retrieved content.
// For each evidence dia_id, looks up the turn text and checks substring match.
// Logs a warning for turns with text < 20 chars (higher false-positive risk).
func RecallHitRate(evidenceIDs []string, sample *LocomoSample, retrievedContent string) float64 {
	if len(evidenceIDs) == 0 {
		return 1.0 // no evidence required = perfect
	}

	// Expand any multi-ID evidence entries (e.g. "D8:6; D9:17" or "D9:1 D4:4")
	var expanded []string
	for _, id := range evidenceIDs {
		split := SplitEvidenceIDs(id)
		if split != nil {
			expanded = append(expanded, split...)
		}
	}
	if len(expanded) == 0 {
		log.Printf("WARNING: no valid dia_ids after expanding evidence %v", evidenceIDs)
		return float64(0) / float64(len(evidenceIDs))
	}

	// Build turn index once (avoids re-parsing JSON per ID)
	turns := GetTurns(sample)
	turnMap := make(map[string]*LocomoTurn, len(turns))
	for i := range turns {
		turnMap[turns[i].DiaID] = &turns[i]
	}

	lowerRetrieved := strings.ToLower(retrievedContent)
	found := 0
	resolvable := 0
	for _, diaID := range expanded {
		turn, ok := turnMap[diaID]
		if !ok {
			log.Printf("WARNING: dia_id %q not found in sample %s", diaID, sample.SampleID)
			continue
		}
		resolvable++
		if len(turn.Text) < 20 {
			log.Printf("WARNING: short turn text (%d chars) for dia_id %s: %q",
				len(turn.Text), diaID, turn.Text)
		}
		if strings.Contains(lowerRetrieved, strings.ToLower(turn.Text)) {
			found++
		}
	}
	if resolvable == 0 {
		return 0.0 // no resolvable evidence = can't evaluate
	}
	return float64(found) / float64(resolvable)
}

// BudgetTruncate truncates messages to fit within a token budget.
// Returns the truncated messages and total token count.
func BudgetTruncate(messages []string, budgetTokens int) ([]string, int) {
	var result []string
	total := 0
	// Walk from the front (best first) and keep until budget exhausted.
	for i := 0; i < len(messages); i++ {
		tokens := len(messages[i]) / 4
		if total+tokens > budgetTokens && len(result) > 0 {
			break
		}
		result = append(result, messages[i])
		total += tokens
	}
	return result, total
}

// StringListToContent joins a list of strings into a single content string.
func StringListToContent(parts []string) string {
	return strings.Join(parts, "\n")
}
