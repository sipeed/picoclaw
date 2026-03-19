package telegram

import (
	"net/http"
	"sync/atomic"
)

// resilientTransport wraps http.RoundTripper to detect polling failures
// and notify on recovery.
type resilientTransport struct {
	base      http.RoundTripper
	onFailure func() // called once when first failure detected
	onRecover func() // called once when connection recovers after failure
	failed    atomic.Bool
}

func (t *resilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		if !t.failed.Swap(true) && t.onFailure != nil {
			t.onFailure()
		}
	} else if t.failed.Swap(false) && t.onRecover != nil {
		t.onRecover()
	}
	return resp, err
}
