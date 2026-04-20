package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ProcessRequest represents an incoming request to process asynchronously
type ProcessRequest struct {
	WebhookURL string                 `json:"webhook_url"`
	Payload    map[string]interface{} `json:"payload"`
}

// ProcessResponse is returned immediately when a job is accepted
type ProcessResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// WebhookPayload is sent to the webhook URL when processing completes
type WebhookPayload struct {
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Processor handles async processing and webhook callbacks
type Processor struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	httpClient  *http.Client
	processorFn ProcessorFunc
}

// Job tracks the state of an async job
type Job struct {
	ID         string
	WebhookURL string
	Payload    map[string]interface{}
	Status     string
	CreatedAt  time.Time
	CompletedAt *time.Time
}

// ProcessorFunc is the actual processing function to be executed
type ProcessorFunc func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)

// NewProcessor creates a new webhook processor
func NewProcessor(processorFn ProcessorFunc) *Processor {
	return &Processor{
		jobs: make(map[string]*Job),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		processorFn: processorFn,
	}
}

// Submit accepts a new job and returns immediately
func (p *Processor) Submit(req ProcessRequest) (*ProcessResponse, error) {
	if req.WebhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required")
	}

	jobID := uuid.New().String()
	job := &Job{
		ID:         jobID,
		WebhookURL: req.WebhookURL,
		Payload:    req.Payload,
		Status:     "processing",
		CreatedAt:  time.Now(),
	}

	p.mu.Lock()
	p.jobs[jobID] = job
	p.mu.Unlock()

	// Start processing in background
	go p.processJob(job)

	logger.InfoCF("webhook", "Job submitted", map[string]any{
		"job_id":      jobID,
		"webhook_url": req.WebhookURL,
	})

	return &ProcessResponse{
		JobID:     jobID,
		Status:    "processing",
		Timestamp: time.Now(),
	}, nil
}

// GetJob retrieves job status
func (p *Processor) GetJob(jobID string) (*Job, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	job, exists := p.jobs[jobID]
	return job, exists
}

// processJob executes the processing and calls webhook
func (p *Processor) processJob(job *Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.InfoCF("webhook", "Processing job started", map[string]any{
		"job_id": job.ID,
	})

	result, err := p.processorFn(ctx, job.Payload)

	completedAt := time.Now()
	job.CompletedAt = &completedAt

	var webhookPayload WebhookPayload
	if err != nil {
		job.Status = "failed"
		webhookPayload = WebhookPayload{
			JobID:     job.ID,
			Status:    "failed",
			Error:     err.Error(),
			Timestamp: completedAt,
		}
		logger.ErrorCF("webhook", "Job processing failed", map[string]any{
			"job_id": job.ID,
			"error":  err.Error(),
		})
	} else {
		job.Status = "completed"
		webhookPayload = WebhookPayload{
			JobID:     job.ID,
			Status:    "completed",
			Result:    result,
			Timestamp: completedAt,
		}
		logger.InfoCF("webhook", "Job processing completed", map[string]any{
			"job_id": job.ID,
		})
	}

	// Call webhook
	if err := p.callWebhook(job.WebhookURL, webhookPayload); err != nil {
		logger.ErrorCF("webhook", "Webhook callback failed", map[string]any{
			"job_id":      job.ID,
			"webhook_url": job.WebhookURL,
			"error":       err.Error(),
		})
	}
}

// callWebhook sends the result to the webhook URL
func (p *Processor) callWebhook(webhookURL string, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PicoClaw-Webhook/1.0")

	logger.InfoCF("webhook", "Calling webhook", map[string]any{
		"url":    webhookURL,
		"job_id": payload.JobID,
	})

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	logger.InfoCF("webhook", "Webhook called successfully", map[string]any{
		"url":         webhookURL,
		"job_id":      payload.JobID,
		"status_code": resp.StatusCode,
	})

	return nil
}

// CleanupOldJobs removes jobs older than the specified duration
func (p *Processor) CleanupOldJobs(maxAge time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, job := range p.jobs {
		if job.CreatedAt.Before(cutoff) {
			delete(p.jobs, id)
		}
	}
}
