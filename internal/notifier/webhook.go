package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type WebhookNotifier struct {
	url     string
	headers map[string]string
	client  *http.Client
}

func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		url:     url,
		headers: headers,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (w *WebhookNotifier) Notify(report domain.ForensicReport) error {
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("POST", w.url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		for k, v := range w.headers {
			req.Header.Set(k, v)
		}

		resp, err := w.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send webhook: %w", err)
		} else {
			if err := drainAndClose(resp); err != nil {
				lastErr = fmt.Errorf("failed to read webhook response: %w", err)
			} else {
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}

				lastErr = fmt.Errorf("webhook returned status: %d", resp.StatusCode)
				if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
					return lastErr
				}
			}
		}

		if attempt < 2 {
			time.Sleep(backoff(attempt))
		}
	}

	return lastErr
}

func (w *WebhookNotifier) Name() string {
	return "webhook"
}
