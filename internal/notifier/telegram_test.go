package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func TestSlackNotifier_Name(t *testing.T) {
	notifier := NewSlackNotifier("http://example.com", "#alerts")

	if name := notifier.Name(); name != "slack" {
		t.Errorf("Name() = %v, want slack", name)
	}
}

func TestSlackNotifier_colorForReason(t *testing.T) {
	notifier := NewSlackNotifier("", "")

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

func TestSlackNotifier_Notify_Success(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, "#alerts")

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

	var msg slackMessage
	if err := json.Unmarshal(receivedBody, &msg); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if msg.Channel != "#alerts" {
		t.Errorf("Channel = %v, want #alerts", msg.Channel)
	}
	if !strings.Contains(msg.Text, "Pod Crash Detected") {
		t.Errorf("Text should contain 'Pod Crash Detected', got: %s", msg.Text)
	}
	if len(msg.Attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(msg.Attachments))
	}
	if msg.Attachments[0].Color != "danger" {
		t.Errorf("Color = %v, want danger (for OOMKilled)", msg.Attachments[0].Color)
	}
}

func TestSlackNotifier_Notify_MessageFormat(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, "")

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "app",
		Reason:        "Error",
		ExitCode:      1,
	}
	report := *domain.NewForensicReport(crash)

	notifier.Notify(report)

	var msg slackMessage
	json.Unmarshal(receivedBody, &msg)

	fieldMap := make(map[string]string)
	for _, field := range msg.Attachments[0].Fields {
		fieldMap[field.Title] = field.Value
	}

	if fieldMap["Namespace"] != "default" {
		t.Errorf("Namespace field = %v, want default", fieldMap["Namespace"])
	}
	if fieldMap["Pod"] != "test-pod" {
		t.Errorf("Pod field = %v, want test-pod", fieldMap["Pod"])
	}
	if fieldMap["Exit Code"] != "1" {
		t.Errorf("Exit Code field = %v, want 1", fieldMap["Exit Code"])
	}
	if fieldMap["Report ID"] != report.ID {
		t.Errorf("Report ID field = %v, want %s", fieldMap["Report ID"], report.ID)
	}
}

func TestSlackNotifier_Notify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, "")
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestSlackNotifier_Notify_ConnectionError(t *testing.T) {
	notifier := NewSlackNotifier("http://localhost:99999", "")
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for connection failure")
	}
}

func TestSlackNotifier_ImplementsNotifierInterface(t *testing.T) {
	var _ Notifier = (*SlackNotifier)(nil)
}

func TestSlackNotifier_EmptyChannel(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, "")
	report := *domain.NewForensicReport(domain.PodCrash{})

	notifier.Notify(report)

	var msg slackMessage
	json.Unmarshal(receivedBody, &msg)

	if msg.Channel != "" {
		t.Errorf("Channel should be empty when not set, got: %s", msg.Channel)
	}
}

func BenchmarkSlackNotifier_Notify(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL, "#alerts")
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
