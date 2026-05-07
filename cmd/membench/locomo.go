package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LocomoSample represents one conversation sample from the LOCOMO dataset.
type LocomoSample struct {
	SampleID     string                     `json:"sample_id"`
	Conversation map[string]json.RawMessage `json:"conversation"`
	QA           []LocomoQA                 `json:"qa"`
}

// LocomoTurn represents a single turn in a conversation.
type LocomoTurn struct {
	Speaker string `json:"speaker"`
	DiaID   string `json:"dia_id"`
	Text    string `json:"text"`
}

// LocomoQA represents a question-answer pair with evidence.
type LocomoQA struct {
	Question          string          `json:"question"`
	Answer            json.RawMessage `json:"answer"`             // can be string or int (category 1-4)
	AdversarialAnswer string          `json:"adversarial_answer"` // category 5 only
	Evidence          []string        `json:"evidence"`
	Category          int             `json:"category"` // 1=single-hop, 2=multi-hop, 3=open-ended, 5=adversarial
}

// AnswerString returns the answer as a string, handling both string and int types.
func (qa *LocomoQA) AnswerString() string {
	// Prefer answer field (category 1-4)
	if len(qa.Answer) > 0 {
		var s string
		if err := json.Unmarshal(qa.Answer, &s); err == nil {
			return s
		}
		var n json.Number
		if err := json.Unmarshal(qa.Answer, &n); err == nil {
			return n.String()
		}
		return strings.Trim(string(qa.Answer), `"`)
	}
	// Fallback to adversarial_answer (category 5)
	return qa.AdversarialAnswer
}

// LoadDataset reads all JSON files from dataDir and returns parsed samples.
func LoadDataset(dataDir string) ([]LocomoSample, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("read data dir %s: %w", dataDir, err)
	}

	var samples []LocomoSample
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(dataDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read file %s: %w", path, err)
			}
			var batch []LocomoSample
			if err := json.Unmarshal(data, &batch); err != nil {
				return nil, fmt.Errorf("parse file %s: %w", path, err)
			}
			samples = append(samples, batch...)
		}
	}
	return samples, nil
}

// GetSessionNames returns sorted session keys (session_1, session_2, ...) from conversation.
func GetSessionNames(conv map[string]json.RawMessage) []string {
	var names []string
	for k := range conv {
		if strings.HasPrefix(k, "session_") && !strings.Contains(k, "_date_time") {
			names = append(names, k)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		ni := sessionNum(names[i])
		nj := sessionNum(names[j])
		return ni < nj
	})
	return names
}

func sessionNum(key string) int {
	// "session_1" → 1, "session_10" → 10
	parts := strings.SplitN(key, "_", 2)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[1])
	return n
}

// GetTurns flattens all sessions' turns in chronological order.
func GetTurns(sample *LocomoSample) []LocomoTurn {
	names := GetSessionNames(sample.Conversation)
	var all []LocomoTurn
	for _, name := range names {
		raw, ok := sample.Conversation[name]
		if !ok {
			continue
		}
		var turns []LocomoTurn
		if err := json.Unmarshal(raw, &turns); err != nil {
			log.Printf("WARNING: unmarshal failed for session %q in sample %s: %v", name, sample.SampleID, err)
			continue
		}
		all = append(all, turns...)
	}
	return all
}

// GetTurnByDiaID finds a specific turn by dia_id (e.g. "D1:3").
func GetTurnByDiaID(sample *LocomoSample, diaID string) *LocomoTurn {
	turns := GetTurns(sample)
	for i := range turns {
		if turns[i].DiaID == diaID {
			return &turns[i]
		}
	}
	return nil
}

// GetSpeakers returns the two speaker names from conversation metadata.
func GetSpeakers(conv map[string]json.RawMessage) (string, string) {
	var a, b string
	json.Unmarshal(conv["speaker_a"], &a)
	json.Unmarshal(conv["speaker_b"], &b)
	return a, b
}
