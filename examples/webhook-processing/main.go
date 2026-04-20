package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/webhook"
)

// CustomProcessor demonstrates a custom processing function
func CustomProcessor(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	// Simulate processing that takes some time
	processingTime := 3 * time.Second

	log.Printf("Starting to process payload: %+v", payload)

	select {
	case <-time.After(processingTime):
		// Processing completed
	case <-ctx.Done():
		return nil, fmt.Errorf("processing cancelled: %w", ctx.Err())
	}

	// Extract and process data
	data, ok := payload["data"]
	if !ok {
		return nil, fmt.Errorf("missing 'data' field in payload")
	}

	// Perform your custom processing here
	processedResult := fmt.Sprintf("Processed: %v", data)

	result := map[string]interface{}{
		"original_data":   data,
		"processed_data":  processedResult,
		"processed_at":    time.Now().Format(time.RFC3339),
		"processing_time": processingTime.String(),
		"status":          "success",
	}

	log.Printf("Processing completed: %+v", result)
	return result, nil
}

func main() {
	// Create processor with custom processing function
	processor := webhook.NewProcessor(CustomProcessor)

	// Create HTTP handler with optional auth token
	authToken := "your-secret-token" // In production, load from env or config
	handler := webhook.NewHandler(processor, authToken)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/process", handler.ProcessHandler)
	mux.HandleFunc("/webhook/status", handler.StatusHandler)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			processor.CleanupOldJobs(1 * time.Hour)
			log.Println("Cleaned up old jobs")
		}
	}()

	// Start server
	addr := ":8080"
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Starting webhook processing server on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  POST   /webhook/process - Submit processing job")
	log.Printf("  GET    /webhook/status  - Check job status")
	log.Printf("  GET    /health          - Health check")
	log.Printf("\nAuth token: %s", authToken)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
