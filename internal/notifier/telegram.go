package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

const (
	telegramAPIBaseURL = "https://api.telegram.org/bot"
)

type TelegramNotifier struct {
	webhookURL string
	token      string
	chatId     string
}

type telegramMessage struct {
	Token  string `json:"token,omitempty"`
	ChatId string `json:"chat_id,omitempty"`
	Text   string `json:"text"`
}

type telegramAttachment struct {
	Color  string          `json:"color"`
	Fields []telegramField `json:"fields"`
}

type telegramField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewTelegramNotifier(webHookURL *string, token, chatId string) *TelegramNotifier {
	if webHookURL == nil {
		webHookURL = new(string)
		*webHookURL = telegramAPIBaseURL
	}
	return &TelegramNotifier{
		webhookURL: *webHookURL,
		chatId:     chatId,
		token:      token,
	}
}

func (s *TelegramNotifier) Notify(report domain.ForensicReport) error {
	msg := telegramMessage{
		ChatId: s.chatId,
		Text: fmt.Sprintf("🚨 *Pod Crash Detected: %s* \n %v", report.Summary(), []telegramAttachment{{
			Color: s.colorForReason(report.Crash.Reason),
			Fields: []telegramField{
				{Title: "Namespace", Value: report.Crash.Namespace, Short: true},
				{Title: "Pod", Value: report.Crash.PodName, Short: true},
				{Title: "Container", Value: report.Crash.ContainerName, Short: true},
				{Title: "Reason", Value: report.Crash.Reason, Short: true},
				{Title: "Exit Code", Value: fmt.Sprintf("%d", report.Crash.ExitCode), Short: true},
				{Title: "Restart Count", Value: fmt.Sprintf("%d", report.Crash.RestartCount), Short: true},
				{Title: "Report ID", Value: report.ID, Short: false},
				{Title: "Collected", Value: report.CollectedAt.Format("2006-01-02 15:04:05"), Short: true},
			},
		}}),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s%s/sendMessage", s.webhookURL, s.token), "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send telegram notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

func (s *TelegramNotifier) Name() string {
	return "telegram"
}

func (s *TelegramNotifier) colorForReason(reason string) string {
	switch reason {
	case "OOMKilled":
		return "danger"
	case "CrashLoopBackOff":
		return "warning"
	default:
		return "#ff9500"
	}
}
