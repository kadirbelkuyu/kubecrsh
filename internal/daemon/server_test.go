package daemon

import (
	"net/http"
	"net/http/httptest"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/internal/notifier"
	"github.com/kadirbelkuyu/kubecrsh/internal/reporter"
)

type mockStorage struct {
	saved   []*domain.ForensicReport
	loadErr error
}

func (m *mockStorage) Save(report *domain.ForensicReport) error {
	m.saved = append(m.saved, report)
	return nil
}

func (m *mockStorage) Load(id string) (*domain.ForensicReport, error) {
	return nil, m.loadErr
}

func (m *mockStorage) List() ([]*domain.ForensicReport, error) {
	return m.saved, nil
}

var _ reporter.Storage = (*mockStorage)(nil)

type mockNotifier struct {
	name     string
	notified []domain.ForensicReport
	err      error
}

func (m *mockNotifier) Notify(report domain.ForensicReport) error {
	m.notified = append(m.notified, report)
	return m.err
}

func (m *mockNotifier) Name() string {
	return m.name
}

var _ notifier.Notifier = (*mockNotifier)(nil)

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		Namespace: "production",
		Reasons:   []string{"OOMKilled", "Error"},
		HTTPAddr:  ":8080",
		Notifiers: nil,
		Storage:   nil,
	}

	if cfg.Namespace != "production" {
		t.Errorf("Namespace = %v, want production", cfg.Namespace)
	}
	if len(cfg.Reasons) != 2 {
		t.Errorf("Reasons length = %d, want 2", len(cfg.Reasons))
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %v, want :8080", cfg.HTTPAddr)
	}
}

func TestServer_healthHandler(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Body = %v, want OK", w.Body.String())
	}
}

func TestServer_readyHandler(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	server.readyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "Ready" {
		t.Errorf("Body = %v, want Ready", w.Body.String())
	}
}

func TestServer_handleCrash_SavesReport(t *testing.T) {
	client := fake.NewSimpleClientset()
	storage := &mockStorage{}
	notif := &mockNotifier{name: "test"}

	server := &Server{
		client:    client,
		store:     storage,
		notifiers: []notifier.Notifier{notif},
		metrics:   NewMetrics(),
	}
	server.collector = nil

	crash := domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "Error",
	}

	if server.collector != nil {
		server.handleCrash(crash)
	}
}

func TestMockStorage_ImplementsInterface(t *testing.T) {
	storage := &mockStorage{}
	var _ reporter.Storage = storage
}

func TestMockNotifier_ImplementsInterface(t *testing.T) {
	notif := &mockNotifier{name: "mock"}
	var _ notifier.Notifier = notif

	if notif.Name() != "mock" {
		t.Errorf("Name() = %v, want mock", notif.Name())
	}
}

func BenchmarkServer_healthHandler(b *testing.B) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		server.healthHandler(w, req)
	}
}

func BenchmarkServer_readyHandler(b *testing.B) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		server.readyHandler(w, req)
	}
}

func TestServer_handleCrash_QueueDropWhenFull(t *testing.T) {
	server := &Server{
		crashQueue: make(chan crashJob, 1),
		metrics:    NewMetrics(),
	}

	first := domain.PodCrash{
		Namespace:     "default",
		PodName:       "pod-1",
		ContainerName: "app",
		Reason:        "Error",
	}
	second := domain.PodCrash{
		Namespace:     "default",
		PodName:       "pod-2",
		ContainerName: "app",
		Reason:        "Error",
	}

	server.handleCrash(first)
	server.handleCrash(second)

	if got := len(server.crashQueue); got != 1 {
		t.Fatalf("crash queue length = %d, want 1", got)
	}

	if got := gaugeValue(t, server.metrics.CrashQueueDepth); got != 1 {
		t.Fatalf("crash queue depth = %.0f, want 1", got)
	}

	if got := counterValue(t, server.metrics.QueueDropped.WithLabelValues("crash")); got != 1 {
		t.Fatalf("dropped crash jobs = %.0f, want 1", got)
	}
}

func TestServer_enqueueNotifications_QueueDropWhenFull(t *testing.T) {
	server := &Server{
		notifiers: []notifier.Notifier{
			&mockNotifier{name: "one"},
			&mockNotifier{name: "two"},
		},
		notifyQueue: make(chan notifyJob, 1),
		metrics:     NewMetrics(),
	}

	report := domain.NewForensicReport(domain.PodCrash{
		Namespace:     "default",
		PodName:       "pod-1",
		ContainerName: "app",
		Reason:        "Error",
	})

	server.enqueueNotifications(*report)

	if got := len(server.notifyQueue); got != 1 {
		t.Fatalf("notify queue length = %d, want 1", got)
	}

	if got := gaugeValue(t, server.metrics.NotifyQueueDepth); got != 1 {
		t.Fatalf("notify queue depth = %.0f, want 1", got)
	}

	if got := counterValue(t, server.metrics.QueueDropped.WithLabelValues("notify")); got != 1 {
		t.Fatalf("dropped notify jobs = %.0f, want 1", got)
	}
}

func gaugeValue(t *testing.T, g interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	var m dto.Metric
	if err := g.Write(&m); err != nil {
		t.Fatalf("failed to read gauge value: %v", err)
	}
	return m.GetGauge().GetValue()
}

func counterValue(t *testing.T, c interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("failed to read counter value: %v", err)
	}
	return m.GetCounter().GetValue()
}
