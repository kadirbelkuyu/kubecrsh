package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type SlackNotifier struct {
	webhookURL string
	channel    string
	client     *http.Client
}

type slackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string       `json:"color"`
	Fields []slackField `json:"fields"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlackNotifier(webhookURL, channel string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SlackNotifier) Notify(report domain.ForensicReport) error {
	msg := slackMessage{
		Channel: s.channel,
		Text:    fmt.Sprintf("🚨 *Pod Crash Detected: %s*", report.Summary()),
		Attachments: []slackAttachment{{
			Color: s.colorForReason(report.Crash.Reason),
			Fields: []slackField{
				{Title: "Namespace", Value: report.Crash.Namespace, Short: true},
				{Title: "Pod", Value: report.Crash.PodName, Short: true},
				{Title: "Container", Value: report.Crash.ContainerName, Short: true},
				{Title: "Reason", Value: report.Crash.Reason, Short: true},
				{Title: "Exit Code", Value: fmt.Sprintf("%d", report.Crash.ExitCode), Short: true},
				{Title: "Restart Count", Value: fmt.Sprintf("%d", report.Crash.RestartCount), Short: true},
				{Title: "Report ID", Value: report.ID, Short: false},
				{Title: "Collected", Value: report.CollectedAt.Format("2006-01-02 15:04:05"), Short: true},
			},
		}},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("POST", s.webhookURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send slack notification: %w", err)
		} else {
			if err := drainAndClose(resp); err != nil {
				lastErr = fmt.Errorf("failed to read slack response: %w", err)
			} else {
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}

				lastErr = fmt.Errorf("slack returned non-2xx status: %d", resp.StatusCode)
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

func (s *SlackNotifier) Name() string {
	return "slack"
}

func (s *SlackNotifier) colorForReason(reason string) string {
	switch reason {
	case "OOMKilled":
		return "danger"
	case "CrashLoopBackOff":
		return "warning"
	default:
		return "#ff9500"
	}
}
