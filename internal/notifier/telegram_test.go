package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

var (
	webHookURL = "#tests"
	token      = "bot_id:token"
	chatId     = "chat_id"
)

func init() {

}

func TestTelegramNotifier_Name(t *testing.T) {
	notifier := NewTelegramNotifier(&webHookURL, "#alerts", "")

	if name := notifier.Name(); name != "telegram" {
		t.Errorf("Name() = %v, want slack", name)
	}
}

func TestTelegramNotifier_colorForReason(t *testing.T) {
	notifier := NewTelegramNotifier(&webHookURL, "", "")

	tests := []struct {
		reason string
		want   string
	}{
		{"OOMKilled", "danger"},
		{"CrashLoopBackOff", "warning"},
		{"Error", "#ff9500"},
		{"Unknown", "#ff9500"},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if got := notifier.colorForReason(tt.reason); got != tt.want {
				t.Errorf("colorForReason(%s) = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}

func TestTelegramNotifier_Notify_Success(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&server.URL, token, chatId)

	crash := domain.PodCrash{
		Namespace:     "production",
		PodName:       "api-server",
		ContainerName: "main",
		Reason:        "OOMKilled",
		ExitCode:      137,
		RestartCount:  5,
	}
	report := *domain.NewForensicReport(crash)

	err := notifier.Notify(report)
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

}

func TestTelegramNotifier_Notify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&webHookURL, "", "")
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestTelegramNotifier_Notify_ConnectionError(t *testing.T) {
	notifier := NewTelegramNotifier(&webHookURL, "", "")
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for connection failure")
	}
}

func TestTelegramNotifier_ImplementsNotifierInterface(t *testing.T) {
	var _ Notifier = (*TelegramNotifier)(nil)
}

func TestTelegramNotifier_EmptyChannel(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&webHookURL, "", "")
	report := *domain.NewForensicReport(domain.PodCrash{})

	notifier.Notify(report)

	var msg slackMessage
	json.Unmarshal(receivedBody, &msg)

	if msg.Channel != "" {
		t.Errorf("Channel should be empty when not set, got: %s", msg.Channel)
	}
}

func BenchmarkTelegramNotifier_Notify(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&webHookURL, "#alerts", "")
	report := *domain.NewForensicReport(domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "Error",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		notifier.Notify(report)
	}
}
