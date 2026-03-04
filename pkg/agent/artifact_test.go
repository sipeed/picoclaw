package agent

import (
	"testing"
	"time"
)

// TestArtifact tests basic Artifact functionality
func TestArtifact(t *testing.T) {
	t.Run("ToMessage", func(t *testing.T) {
		artifact := &Artifact{
			ID:        "artifact_123",
			Type:      ArtifactTypeCode,
			Content:   "package main",
			Size:      12,
			Created:   time.Now(),
			SessionID: "session_1",
		}

		msg := artifact.ToMessage()

		if msg.Type != MessageTypeArtifact {
			t.Errorf("Expected type 'artifact', got '%s'", msg.Type)
		}
		if msg.ArtifactID != artifact.ID {
			t.Error("Artifact ID mismatch")
		}
		if msg.Content != artifact.Content {
			t.Error("Content mismatch")
		}
	})

	t.Run("BuilderPattern", func(t *testing.T) {
		artifact := &Artifact{ID: "test"}
		artifact.
			WithLanguage("go").
			WithFilename("main.go").
			WithDescription("Main entry point").
			WithMIME("text/x-go").
			WithMetadata("version", "1.0")

		if artifact.Language != "go" {
			t.Error("Language not set")
		}
		if artifact.Filename != "main.go" {
			t.Error("Filename not set")
		}
		if artifact.Description != "Main entry point" {
			t.Error("Description not set")
		}
		if artifact.MIME != "text/x-go" {
			t.Error("MIME not set")
		}
		if artifact.Metadata["version"] != "1.0" {
			t.Error("Metadata not set")
		}
	})

	t.Run("Clone", func(t *testing.T) {
		original := &Artifact{
			ID:       "test",
			Content:  "original",
			Metadata: map[string]any{"key": "value"},
		}

		cloned := original.Clone()
		cloned.Content = "modified"
		cloned.Metadata["key"] = "modified"

		if original.Content != "original" {
			t.Error("Original content was modified")
		}
		if original.Metadata["key"] != "value" {
			t.Error("Original metadata was modified")
		}
	})
}

// TestArtifactManager tests ArtifactManager functionality
func TestArtifactManager(t *testing.T) {
	t.Run("CreateArtifact", func(t *testing.T) {
		am := NewArtifactManager()
		artifact := am.CreateArtifact(ArtifactTypeCode, "package main", "session_1")

		if artifact.ID == "" {
			t.Error("Expected artifact to have an ID")
		}
		if artifact.Type != ArtifactTypeCode {
			t.Error("Artifact type mismatch")
		}
		if artifact.SessionID != "session_1" {
			t.Error("Session ID mismatch")
		}
		if artifact.Size != 12 {
			t.Errorf("Expected size 12, got %d", artifact.Size)
		}

		// Verify it's stored
		retrieved, ok := am.GetArtifact(artifact.ID)
		if !ok {
			t.Error("Artifact not found in manager")
		}
		if retrieved.ID != artifact.ID {
			t.Error("Retrieved artifact ID mismatch")
		}
	})

	t.Run("AddArtifact", func(t *testing.T) {
		am := NewArtifactManager()
		artifact := &Artifact{
			ID:        "custom_id",
			Type:      ArtifactTypeImage,
			Content:   "image data",
			SessionID: "session_1",
		}

		am.AddArtifact(artifact)

		retrieved, ok := am.GetArtifact("custom_id")
		if !ok {
			t.Error("Artifact not found")
		}
		if retrieved.Type != ArtifactTypeImage {
			t.Error("Type mismatch")
		}
	})

	t.Run("GetSessionArtifacts", func(t *testing.T) {
		am := NewArtifactManager()
		am.CreateArtifact(ArtifactTypeCode, "code1", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code2", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code3", "session_2")

		artifacts := am.GetSessionArtifacts("session_1")
		if len(artifacts) != 2 {
			t.Errorf("Expected 2 artifacts for session_1, got %d", len(artifacts))
		}

		artifacts = am.GetSessionArtifacts("session_2")
		if len(artifacts) != 1 {
			t.Errorf("Expected 1 artifact for session_2, got %d", len(artifacts))
		}

		artifacts = am.GetSessionArtifacts("nonexistent")
		if artifacts != nil {
			t.Error("Expected nil for nonexistent session")
		}
	})

	t.Run("GetRecentArtifacts", func(t *testing.T) {
		am := NewArtifactManager()
		for i := 0; i < 5; i++ {
			am.CreateArtifact(ArtifactTypeCode, "code", "session_1")
			time.Sleep(time.Millisecond) // Ensure different timestamps
		}

		recent := am.GetRecentArtifacts("session_1", 3)
		if len(recent) != 3 {
			t.Errorf("Expected 3 recent artifacts, got %d", len(recent))
		}

		// Should return all if limit exceeds count
		recent = am.GetRecentArtifacts("session_1", 10)
		if len(recent) != 5 {
			t.Errorf("Expected 5 artifacts, got %d", len(recent))
		}
	})

	t.Run("GetArtifactsByType", func(t *testing.T) {
		am := NewArtifactManager()
		am.CreateArtifact(ArtifactTypeCode, "code", "session_1")
		am.CreateArtifact(ArtifactTypeImage, "image", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code2", "session_1")

		codeArtifacts := am.GetArtifactsByType("session_1", ArtifactTypeCode)
		if len(codeArtifacts) != 2 {
			t.Errorf("Expected 2 code artifacts, got %d", len(codeArtifacts))
		}

		imageArtifacts := am.GetArtifactsByType("session_1", ArtifactTypeImage)
		if len(imageArtifacts) != 1 {
			t.Errorf("Expected 1 image artifact, got %d", len(imageArtifacts))
		}
	})

	t.Run("UpdateArtifact", func(t *testing.T) {
		am := NewArtifactManager()
		artifact := am.CreateArtifact(ArtifactTypeCode, "original", "session_1")

		err := am.UpdateArtifact(artifact.ID, func(a *Artifact) {
			a.Content = "updated"
			a.Language = "go"
		})
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		updated, _ := am.GetArtifact(artifact.ID)
		if updated.Content != "updated" {
			t.Error("Content not updated")
		}
		if updated.Language != "go" {
			t.Error("Language not updated")
		}
	})

	t.Run("UpdateArtifact_NotFound", func(t *testing.T) {
		am := NewArtifactManager()
		err := am.UpdateArtifact("nonexistent", func(a *Artifact) {})

		if err == nil {
			t.Error("Expected error for nonexistent artifact")
		}
	})

	t.Run("DeleteArtifact", func(t *testing.T) {
		am := NewArtifactManager()
		artifact := am.CreateArtifact(ArtifactTypeCode, "code", "session_1")

		err := am.DeleteArtifact(artifact.ID)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		_, ok := am.GetArtifact(artifact.ID)
		if ok {
			t.Error("Artifact still exists after deletion")
		}

		// Verify it's removed from session index
		artifacts := am.GetSessionArtifacts("session_1")
		if len(artifacts) != 0 {
			t.Error("Artifact still in session index")
		}
	})

	t.Run("DeleteArtifact_NotFound", func(t *testing.T) {
		am := NewArtifactManager()
		err := am.DeleteArtifact("nonexistent")

		if err == nil {
			t.Error("Expected error for nonexistent artifact")
		}
	})

	t.Run("DeleteSessionArtifacts", func(t *testing.T) {
		am := NewArtifactManager()
		am.CreateArtifact(ArtifactTypeCode, "code1", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code2", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code3", "session_2")

		count := am.DeleteSessionArtifacts("session_1")
		if count != 2 {
			t.Errorf("Expected 2 artifacts deleted, got %d", count)
		}

		artifacts := am.GetSessionArtifacts("session_1")
		if len(artifacts) != 0 {
			t.Error("Session still has artifacts after deletion")
		}

		// session_2 should be unaffected
		artifacts = am.GetSessionArtifacts("session_2")
		if len(artifacts) != 1 {
			t.Error("Other session was affected by deletion")
		}
	})

	t.Run("CountArtifacts", func(t *testing.T) {
		am := NewArtifactManager()
		if am.CountArtifacts() != 0 {
			t.Error("Expected 0 artifacts initially")
		}

		am.CreateArtifact(ArtifactTypeCode, "code1", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code2", "session_2")

		if am.CountArtifacts() != 2 {
			t.Errorf("Expected 2 artifacts, got %d", am.CountArtifacts())
		}
	})

	t.Run("CountSessionArtifacts", func(t *testing.T) {
		am := NewArtifactManager()
		am.CreateArtifact(ArtifactTypeCode, "code1", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code2", "session_1")

		count := am.CountSessionArtifacts("session_1")
		if count != 2 {
			t.Errorf("Expected 2 artifacts for session_1, got %d", count)
		}

		count = am.CountSessionArtifacts("nonexistent")
		if count != 0 {
			t.Errorf("Expected 0 artifacts for nonexistent session, got %d", count)
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		am := NewArtifactManager()
		am.CreateArtifact(ArtifactTypeCode, "code", "session_1")
		am.CreateArtifact(ArtifactTypeImage, "image", "session_1")
		am.CreateArtifact(ArtifactTypeCode, "code2", "session_2")

		stats := am.GetStats()

		if stats.Total != 3 {
			t.Errorf("Expected 3 total artifacts, got %d", stats.Total)
		}

		if stats.ByType[ArtifactTypeCode] != 2 {
			t.Errorf("Expected 2 code artifacts, got %d", stats.ByType[ArtifactTypeCode])
		}

		if stats.ByType[ArtifactTypeImage] != 1 {
			t.Errorf("Expected 1 image artifact, got %d", stats.ByType[ArtifactTypeImage])
		}

		if stats.BySession["session_1"] != 2 {
			t.Errorf("Expected 2 artifacts in session_1, got %d", stats.BySession["session_1"])
		}

		if stats.TotalSize == 0 {
			t.Error("Expected non-zero total size")
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		am := NewArtifactManager()
		done := make(chan bool)

		// Concurrent writes
		for i := 0; i < 10; i++ {
			go func(n int) {
				am.CreateArtifact(ArtifactTypeCode, "code", "session_1")
				done <- true
			}(i)
		}

		// Wait for all writes
		for i := 0; i < 10; i++ {
			<-done
		}

		count := am.CountSessionArtifacts("session_1")
		if count != 10 {
			t.Errorf("Expected 10 artifacts, got %d (concurrent writes failed)", count)
		}
	})
}

// TestArtifactFilters tests artifact filtering
func TestArtifactFilters(t *testing.T) {
	now := time.Now()
	artifacts := []*Artifact{
		{Type: ArtifactTypeCode, Language: "go", Size: 100, Created: now.Add(-time.Hour)},
		{Type: ArtifactTypeImage, Size: 500, Created: now.Add(-time.Minute)},
		{Type: ArtifactTypeCode, Language: "python", Size: 200, Created: now},
	}

	t.Run("FilterByType", func(t *testing.T) {
		result := FilterArtifacts(artifacts, FilterArtifactsByType(ArtifactTypeCode))
		if len(result) != 2 {
			t.Errorf("Expected 2 code artifacts, got %d", len(result))
		}
	})

	t.Run("FilterByLanguage", func(t *testing.T) {
		result := FilterArtifacts(artifacts, FilterArtifactsByLanguage("go"))
		if len(result) != 1 {
			t.Errorf("Expected 1 go artifact, got %d", len(result))
		}
	})

	t.Run("FilterByMinSize", func(t *testing.T) {
		result := FilterArtifacts(artifacts, FilterArtifactsByMinSize(150))
		if len(result) != 2 {
			t.Errorf("Expected 2 artifacts >= 150 bytes, got %d", len(result))
		}
	})

	t.Run("FilterSince", func(t *testing.T) {
		result := FilterArtifacts(artifacts, FilterArtifactsSince(now.Add(-30*time.Minute)))
		if len(result) != 2 {
			t.Errorf("Expected 2 recent artifacts, got %d", len(result))
		}
	})
}

// TestGenerateArtifactID tests ID generation
func TestGenerateArtifactID(t *testing.T) {
	t.Run("DeterministicGeneration", func(t *testing.T) {
		content := "test content"
		id1 := generateArtifactID(content)
		id2 := generateArtifactID(content)

		if id1 != id2 {
			t.Error("Expected same ID for same content")
		}
	})

	t.Run("UniqueForDifferentContent", func(t *testing.T) {
		id1 := generateArtifactID("content1")
		id2 := generateArtifactID("content2")

		if id1 == id2 {
			t.Error("Expected different IDs for different content")
		}
	})

	t.Run("HasPrefix", func(t *testing.T) {
		id := generateArtifactID("test")
		if id[:9] != "artifact_" {
			t.Error("Expected artifact ID to have 'artifact_' prefix")
		}
	})
}

// BenchmarkArtifactManager benchmarks artifact manager operations
func BenchmarkArtifactManager(b *testing.B) {
	b.Run("CreateArtifact", func(b *testing.B) {
		am := NewArtifactManager()
		for i := 0; i < b.N; i++ {
			am.CreateArtifact(ArtifactTypeCode, "code content", "session_1")
		}
	})

	b.Run("GetArtifact", func(b *testing.B) {
		am := NewArtifactManager()
		artifact := am.CreateArtifact(ArtifactTypeCode, "code", "session_1")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = am.GetArtifact(artifact.ID)
		}
	})

	b.Run("GetSessionArtifacts", func(b *testing.B) {
		am := NewArtifactManager()
		for i := 0; i < 100; i++ {
			am.CreateArtifact(ArtifactTypeCode, "code", "session_1")
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = am.GetSessionArtifacts("session_1")
		}
	})

	b.Run("FilterArtifacts", func(b *testing.B) {
		artifacts := make([]*Artifact, 100)
		for i := 0; i < 100; i++ {
			artifacts[i] = &Artifact{Type: ArtifactTypeCode}
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = FilterArtifacts(artifacts, FilterArtifactsByType(ArtifactTypeCode))
		}
	})
}
