package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type SlackNotifier struct {
	webhookURL string
	channel    string
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

	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned non-200 status: %d", resp.StatusCode)
	}

	return nil
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
