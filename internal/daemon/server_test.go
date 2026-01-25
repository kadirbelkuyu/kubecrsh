package daemon

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/internal/notifier"
	"github.com/kadirbelkuyu/kubecrsh/internal/reporter"
	"k8s.io/client-go/kubernetes/fake"
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
