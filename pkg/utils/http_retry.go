package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const maxRetries = 3

var retryDelayUnit = time.Second

func shouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode >= 500
}

func DoRequestWithRetry(client *http.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := range maxRetries {
		if i > 0 && resp != nil {
			resp.Body.Close()
		}

		resp, err = client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				break
			}
			if !shouldRetry(resp.StatusCode) {
				break
			}
		}

		if i < maxRetries-1 {
			if err = sleepWithCtx(req.Context(), retryDelayUnit*time.Duration(i+1)); err != nil {
				return nil, fmt.Errorf("failed to sleep: %w", err)
			}
		}
	}
	return resp, err
}

func sleepWithCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
