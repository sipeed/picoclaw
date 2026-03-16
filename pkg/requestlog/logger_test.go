package requestlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if cfg.LogDir != "logs/requests" {
		t.Errorf("expected LogDir 'logs/requests', got %q", cfg.LogDir)
	}
	if cfg.MaxFileSizeMB != 100 {
		t.Errorf("expected MaxFileSizeMB 100, got %d", cfg.MaxFileSizeMB)
	}
	if cfg.RetentionDays != 30 {
		t.Errorf("expected RetentionDays 30, got %d", cfg.RetentionDays)
	}
}

func TestStorage_WriteAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir, 1)

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer storage.Close()

	record := RequestRecord{
		Timestamp: time.Now().UTC(),
		RequestID: "test-123",
		Channel:   "telegram",
		SenderID:  "user1",
		ChatID:    "chat1",
		Content:   "hello world",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if writeErr := storage.Write(data); writeErr != nil {
		t.Fatalf("Write failed: %v", writeErr)
	}

	records, err := storage.Query(QueryOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].RequestID != "test-123" {
		t.Errorf("expected RequestID 'test-123', got %q", records[0].RequestID)
	}
	if records[0].Channel != "telegram" {
		t.Errorf("expected Channel 'telegram', got %q", records[0].Channel)
	}
}

func TestStorage_QueryWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir, 1)

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer storage.Close()

	now := time.Now().UTC()
	records := []RequestRecord{
		{Timestamp: now.Add(-2 * time.Hour), RequestID: "1", Channel: "telegram", SenderID: "user1"},
		{Timestamp: now.Add(-1 * time.Hour), RequestID: "2", Channel: "discord", SenderID: "user2"},
		{Timestamp: now, RequestID: "3", Channel: "telegram", SenderID: "user1"},
	}

	for _, r := range records {
		data, _ := json.Marshal(r)
		if err := storage.Write(data); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	tests := []struct {
		name     string
		opts     QueryOptions
		expected int
	}{
		{
			name:     "filter by channel",
			opts:     QueryOptions{Channel: "telegram", Limit: 10},
			expected: 2,
		},
		{
			name:     "filter by sender",
			opts:     QueryOptions{SenderID: "user2", Limit: 10},
			expected: 1,
		},
		{
			name: "filter by time range",
			opts: QueryOptions{
				StartTime: now.Add(-90 * time.Minute),
				EndTime:   now.Add(10 * time.Minute),
				Limit:     10,
			},
			expected: 2,
		},
		{
			name:     "no filter",
			opts:     QueryOptions{Limit: 10},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.Query(tt.opts)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("expected %d records, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestStorage_QueryWithOffset(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir, 1)

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer storage.Close()

	for i := range 5 {
		record := RequestRecord{
			Timestamp: time.Now().UTC(),
			RequestID: string(rune('a' + i)),
			Channel:   "test",
		}
		data, _ := json.Marshal(record)
		if err := storage.Write(data); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	results, err := storage.Query(QueryOptions{Offset: 2, Limit: 2})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 records, got %d", len(results))
	}
}

func TestStorage_RotateFile(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir, 1)

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	storage.mu.Lock()
	storage.currentSize = storage.maxFileSize + 1
	storage.mu.Unlock()

	record := RequestRecord{Timestamp: time.Now().UTC(), RequestID: "after-rotate"}
	data, _ := json.Marshal(record)

	if err := storage.Write(data); err != nil {
		t.Fatalf("Write after rotate failed: %v", err)
	}

	storage.Close()

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	jsonlCount := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".jsonl" {
			jsonlCount++
		}
	}

	if jsonlCount < 1 {
		t.Errorf("expected at least 1 jsonl file after rotation")
	}
}

func TestReader_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewStorage(tmpDir, 1)

	if err := storage.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer storage.Close()

	now := time.Now().UTC()
	day1 := now.Format("2006-01-02")
	day2 := now.Add(-24 * time.Hour).Format("2006-01-02")

	records := []RequestRecord{
		{Timestamp: now, Channel: "telegram", SenderID: "user1"},
		{Timestamp: now, Channel: "telegram", SenderID: "user2"},
		{Timestamp: now.Add(-24 * time.Hour), Channel: "discord", SenderID: "user1"},
		{Timestamp: now.Add(-24 * time.Hour), Channel: "discord", SenderID: "user3"},
		{Timestamp: now.Add(-24 * time.Hour), Channel: "telegram", SenderID: "user1"},
	}

	for _, r := range records {
		data, _ := json.Marshal(r)
		if err := storage.Write(data); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	reader := NewReader(tmpDir, 1)
	stats, err := reader.GetStats(time.Now().Add(-48*time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats["total"] != 5 {
		t.Errorf("expected total 5, got %v", stats["total"])
	}

	byChannel, ok := stats["by_channel"].(map[string]int)
	if !ok {
		t.Fatal("by_channel is not map[string]int")
	}
	if byChannel["telegram"] != 3 {
		t.Errorf("expected telegram count 3, got %d", byChannel["telegram"])
	}
	if byChannel["discord"] != 2 {
		t.Errorf("expected discord count 2, got %d", byChannel["discord"])
	}

	byDay, ok := stats["by_day"].(map[string]int)
	if !ok {
		t.Fatal("by_day is not map[string]int")
	}
	if byDay[day1] != 2 {
		t.Errorf("expected day1 count 2, got %d", byDay[day1])
	}
	if byDay[day2] != 3 {
		t.Errorf("expected day2 count 3, got %d", byDay[day2])
	}

	topSendersRaw := stats["top_senders"]
	if topSendersRaw == nil {
		t.Fatal("top_senders is nil")
	}

	topSendersData, err := json.Marshal(topSendersRaw)
	if err != nil {
		t.Fatalf("Marshal top_senders failed: %v", err)
	}

	var topSenders []senderStat
	if err := json.Unmarshal(topSendersData, &topSenders); err != nil {
		t.Fatalf("Unmarshal top_senders failed: %v", err)
	}

	if len(topSenders) == 0 {
		t.Error("expected top_senders to have entries")
	}
}

type senderStat struct {
	Sender  string `json:"sender"`
	Channel string `json:"channel"`
	Count   int    `json:"count"`
}
