package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type WebhookNotifier struct {
	url     string
	headers map[string]string
}

func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		url:     url,
		headers: headers,
	}
}

func (w *WebhookNotifier) Notify(report domain.ForensicReport) error {
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	req, err := http.NewRequest("POST", w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

func (w *WebhookNotifier) Name() string {
	return "webhook"
}
