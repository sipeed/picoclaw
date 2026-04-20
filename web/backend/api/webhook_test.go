package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/webhook"
)

func TestHandleWebhookProcess(t *testing.T) {
	h := &Handler{
		configPath: "/tmp/test-config.json",
	}

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "valid request",
			requestBody: `{
				"webhook_url": "https://webhook.site/test",
				"payload": {
					"data": "test data"
				}
			}`,
			expectedStatus: http.StatusAccepted,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp webhook.ProcessResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.JobID == "" {
					t.Error("expected job_id in response")
				}
				if resp.Status != "processing" {
					t.Errorf("expected status 'processing', got %s", resp.Status)
				}
			},
		},
		{
			name: "missing webhook_url",
			requestBody: `{
				"payload": {
					"data": "test"
				}
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/webhook/process", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.handleWebhookProcess(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestHandleWebhookStatus(t *testing.T) {
	h := &Handler{
		configPath: "/tmp/test-config.json",
	}

	// Submit a job first
	processor := h.getWebhookProcessor()
	resp, err := processor.Submit(webhook.ProcessRequest{
		WebhookURL: "https://webhook.site/test",
		Payload: map[string]interface{}{
			"data": "test",
		},
	})
	if err != nil {
		t.Fatalf("failed to submit job: %v", err)
	}

	tests := []struct {
		name           string
		jobID          string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "valid job id",
			jobID:          resp.JobID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var job webhook.Job
				if err := json.NewDecoder(rec.Body).Decode(&job); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if job.ID != resp.JobID {
					t.Errorf("expected job id %s, got %s", resp.JobID, job.ID)
				}
			},
		},
		{
			name:           "non-existent job",
			jobID:          "non-existent-id",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing job_id parameter",
			jobID:          "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/webhook/status"
			if tt.jobID != "" {
				url += "?job_id=" + tt.jobID
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			h.handleWebhookStatus(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestWebhookProcessorInitialization(t *testing.T) {
	h := &Handler{
		configPath: "/tmp/test-config.json",
	}

	// First call should initialize
	processor1 := h.getWebhookProcessor()
	if processor1 == nil {
		t.Fatal("expected processor to be initialized")
	}

	// Second call should return the same instance
	processor2 := h.getWebhookProcessor()
	if processor1 != processor2 {
		t.Error("expected same processor instance")
	}
}

func TestWebhookEndToEnd(t *testing.T) {
	h := &Handler{
		configPath: "/tmp/test-config.json",
	}

	// Create a test server to receive webhooks
	webhookReceived := make(chan webhook.WebhookPayload, 1)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload webhook.WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Logf("failed to decode webhook payload: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		webhookReceived <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Submit job
	reqBody := map[string]interface{}{
		"webhook_url": webhookServer.URL,
		"payload": map[string]interface{}{
			"data": "test data",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/process", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.handleWebhookProcess(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rec.Code)
	}

	var submitResp webhook.ProcessResponse
	if err := json.NewDecoder(rec.Body).Decode(&submitResp); err != nil {
		t.Fatalf("failed to decode submit response: %v", err)
	}

	// Wait for webhook callback (with timeout)
	select {
	case payload := <-webhookReceived:
		if payload.JobID != submitResp.JobID {
			t.Errorf("expected job_id %s, got %s", submitResp.JobID, payload.JobID)
		}
		if payload.Status != "completed" {
			t.Errorf("expected status 'completed', got %s", payload.Status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook callback")
	}

	// Check final status
	statusReq := httptest.NewRequest(http.MethodGet, "/api/webhook/status?job_id="+submitResp.JobID, nil)
	statusRec := httptest.NewRecorder()

	h.handleWebhookStatus(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusRec.Code)
	}

	var job webhook.Job
	if err := json.NewDecoder(statusRec.Body).Decode(&job); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}

	if job.Status != "completed" {
		t.Errorf("expected final status 'completed', got %s", job.Status)
	}
}
