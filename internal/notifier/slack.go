package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	Color     string       `json:"color"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []slackField `json:"fields,omitempty"`
	Footer    string       `json:"footer,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
	MrkdwnIn  []string     `json:"mrkdwn_in,omitempty"`
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
			Color:     s.colorForReason(report.Crash.Reason),
			Title:     "Forensic crash snapshot",
			Text:      slackEvidenceText(report),
			Fields:    slackCrashFields(report),
			Footer:    fmt.Sprintf("kubecrsh · forensic report %s", report.ID),
			Timestamp: slackTimestamp(report.CollectedAt),
			MrkdwnIn:  []string{"text", "fields"},
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

func slackCrashFields(report domain.ForensicReport) []slackField {
	crash := report.Crash

	return []slackField{
		{Title: "Namespace", Value: slackFieldValue(crash.Namespace), Short: true},
		{Title: "Pod", Value: slackFieldValue(crash.PodName), Short: true},
		{Title: "Container", Value: slackFieldValue(crash.ContainerName), Short: true},
		{Title: "Reason", Value: slackFieldValue(crash.Reason), Short: true},
		{Title: "Exit Code", Value: fmt.Sprintf("%d", crash.ExitCode), Short: true},
		{Title: "Signal", Value: slackSignalValue(crash.Signal), Short: true},
		{Title: "Restart Count", Value: fmt.Sprintf("%d", crash.RestartCount), Short: true},
		{Title: "Failure Started", Value: slackTimeValue(crash.StartedAt), Short: true},
		{Title: "Failure Finished", Value: slackTimeValue(crash.FinishedAt), Short: true},
		{Title: "Evidence", Value: fmt.Sprintf("%d logs · %d previous · %d events", len(report.Logs), len(report.PreviousLog), len(report.Events)), Short: true},
		{Title: "Warnings", Value: fmt.Sprintf("%d", slackWarningCount(report)), Short: true},
		{Title: "Report ID", Value: slackFieldValue(report.ID), Short: false},
		{Title: "Collected", Value: slackTimeValue(report.CollectedAt), Short: false},
	}
}

func slackEvidenceText(report domain.ForensicReport) string {
	sections := make([]string, 0, 4)
	if logs := slackLogExcerpt(report.Logs); logs != "" {
		sections = append(sections, "*Latest log excerpt*\n"+logs)
	}
	if events := slackEventExcerpt(report.Events); events != "" {
		sections = append(sections, "*Recent Kubernetes events*\n"+events)
	}
	if previousLogs := slackLogExcerpt(report.PreviousLog); previousLogs != "" {
		sections = append(sections, "*Previous log excerpt*\n"+previousLogs)
	}
	if warnings := slackWarningExcerpt(report.Warnings); warnings != "" {
		sections = append(sections, "*Collection warnings*\n"+warnings)
	}

	if len(sections) == 0 {
		return "*Evidence*\nNo log, previous log, event, or collection warning was captured."
	}

	return strings.Join(sections, "\n\n")
}

func slackLogExcerpt(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	start := len(lines) - 12
	if start < 0 {
		start = 0
	}

	content := strings.Join(lines[start:], "\n")
	return slackCodeBlock(truncateSlackText(content, 1200))
}

func slackEventExcerpt(events []domain.Event) string {
	if len(events) == 0 {
		return ""
	}

	start := len(events) - 4
	if start < 0 {
		start = 0
	}

	lines := make([]string, 0, len(events)-start)
	for _, event := range events[start:] {
		reason := strings.TrimSpace(event.Reason)
		if reason == "" {
			reason = strings.TrimSpace(event.Type)
		}
		if reason == "" {
			reason = "Event"
		}

		message := strings.TrimSpace(strings.ReplaceAll(event.Message, "\n", " "))
		if message == "" {
			message = "No message"
		}

		line := fmt.Sprintf("%s: %s", reason, message)
		if event.Count > 1 {
			line += fmt.Sprintf(" (x%d)", event.Count)
		}
		lines = append(lines, line)
	}

	return slackCodeBlock(truncateSlackText(strings.Join(lines, "\n"), 1200))
}

func slackWarningExcerpt(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}

	start := len(warnings) - 4
	if start < 0 {
		start = 0
	}

	lines := make([]string, 0, len(warnings)-start)
	for _, warning := range warnings[start:] {
		warning = strings.TrimSpace(strings.ReplaceAll(warning, "\n", " "))
		if warning != "" {
			lines = append(lines, warning)
		}
	}
	if len(lines) == 0 {
		return ""
	}

	return slackCodeBlock(truncateSlackText(strings.Join(lines, "\n"), 800))
}

func slackCodeBlock(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "```", "'''")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "```" + value + "```"
}

func truncateSlackText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" || limit <= 0 {
		return value
	}

	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return "…" + string(runes[len(runes)-limit+1:])
}

func slackFieldValue(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if value == "" {
		return "—"
	}

	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	return strings.ReplaceAll(value, ">", "&gt;")
}

func slackSignalValue(signal int32) string {
	if signal == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", signal)
}

func slackTimeValue(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.UTC().Format("2006-01-02 15:04:05 UTC")
}

func slackTimestamp(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.Unix()
}

func slackWarningCount(report domain.ForensicReport) int {
	count := len(report.Warnings)
	for _, event := range report.Events {
		if event.IsWarning() {
			count++
		}
	}
	return count
}
