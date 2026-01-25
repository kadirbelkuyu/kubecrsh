package notifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func TestWebhookNotifier_Name(t *testing.T) {
	notifier := NewWebhookNotifier("http://example.com", nil)

	if name := notifier.Name(); name != "webhook" {
		t.Errorf("Name() = %v, want webhook", name)
	}
}

func TestWebhookNotifier_Notify_Success(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, map[string]string{
		"Authorization": "Bearer token123",
	})

	crash := domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "OOMKilled",
	}
	report := *domain.NewForensicReport(crash)

	err := notifier.Notify(report)
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", receivedHeaders.Get("Content-Type"))
	}
	if receivedHeaders.Get("Authorization") != "Bearer token123" {
		t.Errorf("Authorization header not set correctly")
	}

	var receivedReport domain.ForensicReport
	if err := json.Unmarshal(receivedBody, &receivedReport); err != nil {
		t.Fatalf("Failed to unmarshal received body: %v", err)
	}
	if receivedReport.Crash.PodName != "test-pod" {
		t.Errorf("Received PodName = %v, want test-pod", receivedReport.Crash.PodName)
	}
}

func TestWebhookNotifier_Notify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, nil)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestWebhookNotifier_Notify_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, nil)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for bad request response")
	}
}

func TestWebhookNotifier_Notify_ConnectionError(t *testing.T) {
	notifier := NewWebhookNotifier("http://localhost:99999", nil)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for connection failure")
	}
}

func TestWebhookNotifier_Notify_InvalidURL(t *testing.T) {
	notifier := NewWebhookNotifier("://invalid-url", nil)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestWebhookNotifier_Notify_MultipleHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
		"X-API-Key":       "api-key-123",
		"User-Agent":      "kubecrsh/1.0",
	}
	notifier := NewWebhookNotifier(server.URL, headers)
	report := *domain.NewForensicReport(domain.PodCrash{})

	err := notifier.Notify(report)
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	for key, value := range headers {
		if receivedHeaders.Get(key) != value {
			t.Errorf("Header %s = %v, want %v", key, receivedHeaders.Get(key), value)
		}
	}
}

func TestWebhookNotifier_ImplementsNotifierInterface(t *testing.T) {
	var _ Notifier = (*WebhookNotifier)(nil)
}

func TestWebhookNotifier_Notify_SlowServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, nil)
	report := *domain.NewForensicReport(domain.PodCrash{})

	start := time.Now()
	err := notifier.Notify(report)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if duration < 100*time.Millisecond {
		t.Errorf("Expected request to take at least 100ms, took %v", duration)
	}
}

func BenchmarkWebhookNotifier_Notify(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewWebhookNotifier(server.URL, nil)
	report := *domain.NewForensicReport(domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		notifier.Notify(report)
	}
}
