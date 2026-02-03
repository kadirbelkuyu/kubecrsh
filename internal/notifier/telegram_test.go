package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

const (
	testTelegramToken  = "123:ABC"
	testTelegramChatID = "chat_id"
)

func TestTelegramNotifier_Name(t *testing.T) {
	baseURL := "http://example.com"
	notifier := NewTelegramNotifier(&baseURL, testTelegramToken, testTelegramChatID)

	if name := notifier.Name(); name != "telegram" {
		t.Errorf("Name() = %v, want telegram", name)
	}
}

func TestTelegramNotifier_Notify_Success(t *testing.T) {
	var receivedPath string
	var receivedContentType string
	var received telegramSendMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedContentType = r.Header.Get("Content-Type")

		if r.Method != http.MethodPost {
			t.Fatalf("Method = %s, want POST", r.Method)
		}

		decErr := json.NewDecoder(r.Body).Decode(&received)
		_ = r.Body.Close()
		if decErr != nil {
			t.Fatalf("Failed to decode request body: %v", decErr)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&server.URL, testTelegramToken, testTelegramChatID)

	crash := domain.PodCrash{
		Namespace:     "production",
		PodName:       "api-server",
		ContainerName: "main",
		Reason:        "OOMKilled",
		ExitCode:      137,
		RestartCount:  5,
	}
	report := *domain.NewForensicReport(crash)

	if err := notifier.Notify(report); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	wantPath := "/bot" + testTelegramToken + "/sendMessage"
	if receivedPath != wantPath {
		t.Fatalf("Path = %s, want %s", receivedPath, wantPath)
	}

	if receivedContentType != "application/json" {
		t.Fatalf("Content-Type = %s, want application/json", receivedContentType)
	}

	if received.ChatID != testTelegramChatID {
		t.Fatalf("chat_id = %s, want %s", received.ChatID, testTelegramChatID)
	}

	if !strings.Contains(received.Text, report.ID) {
		t.Fatalf("text does not contain report ID")
	}
}

func TestTelegramNotifier_Notify_ServerError(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"ok":false,"description":"server error"}`))
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&server.URL, testTelegramToken, testTelegramChatID)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for server error response")
	}
	if requests < 1 {
		t.Fatalf("Expected at least 1 request, got %d", requests)
	}
}

func TestTelegramNotifier_Notify_ConnectionError(t *testing.T) {
	baseURL := "http://127.0.0.1:1"
	notifier := NewTelegramNotifier(&baseURL, testTelegramToken, testTelegramChatID)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for connection failure")
	}
}

func TestTelegramNotifier_ImplementsNotifierInterface(t *testing.T) {
	var _ Notifier = (*TelegramNotifier)(nil)
}

func BenchmarkTelegramNotifier_Notify(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	notifier := NewTelegramNotifier(&server.URL, testTelegramToken, testTelegramChatID)
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
