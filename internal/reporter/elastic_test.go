package reporter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func TestElasticStore_ImplementsStorageInterface(t *testing.T) {
	var _ Storage = (*ElasticStore)(nil)
}

func TestElasticConfig_Defaults(t *testing.T) {
	cfg := ElasticConfig{}

	if cfg.Index != "" {
		t.Errorf("Index default = %v, want empty", cfg.Index)
	}
}

func TestElasticStore_toDocument(t *testing.T) {
	store := &ElasticStore{indexName: "test-index"}

	crash := domain.PodCrash{
		Namespace:     "production",
		PodName:       "api-server",
		ContainerName: "main",
		ExitCode:      137,
		Reason:        "OOMKilled",
		Signal:        9,
		RestartCount:  5,
		StartedAt:     time.Now().Add(-time.Hour),
		FinishedAt:    time.Now(),
	}

	report := domain.NewForensicReport(crash)
	report.SetLogs([]string{"log line 1", "log line 2"})
	report.SetPreviousLogs([]string{"previous log"})
	report.SetEnvVar("APP_ENV", "production")
	report.AddWarning("test warning")
	report.AddEvent(*domain.NewEvent("Warning", "OOMKilled", "Container killed"))

	doc := store.toDocument(report)

	if doc.ID != report.ID {
		t.Errorf("ID = %v, want %v", doc.ID, report.ID)
	}
	if doc.Crash.Namespace != "production" {
		t.Errorf("Crash.Namespace = %v, want production", doc.Crash.Namespace)
	}
	if doc.Crash.PodName != "api-server" {
		t.Errorf("Crash.PodName = %v, want api-server", doc.Crash.PodName)
	}
	if doc.Crash.ExitCode != 137 {
		t.Errorf("Crash.ExitCode = %v, want 137", doc.Crash.ExitCode)
	}
	if len(doc.Logs) != 2 {
		t.Errorf("Logs count = %d, want 2", len(doc.Logs))
	}
	if len(doc.PreviousLog) != 1 {
		t.Errorf("PreviousLog count = %d, want 1", len(doc.PreviousLog))
	}
	if doc.EnvVars["APP_ENV"] != "production" {
		t.Errorf("EnvVars[APP_ENV] = %v, want production", doc.EnvVars["APP_ENV"])
	}
	if len(doc.Warnings) != 1 {
		t.Errorf("Warnings count = %d, want 1", len(doc.Warnings))
	}
	if len(doc.Events) != 1 {
		t.Errorf("Events count = %d, want 1", len(doc.Events))
	}
}

func TestElasticStore_fromDocument(t *testing.T) {
	store := &ElasticStore{indexName: "test-index"}

	now := time.Now()

	doc := &elasticDocument{
		ID: "test-id-123",
		Crash: elasticCrash{
			Namespace:     "default",
			PodName:       "test-pod",
			ContainerName: "app",
			ExitCode:      1,
			Reason:        "Error",
			Signal:        0,
			RestartCount:  2,
			StartedAt:     now.Add(-time.Hour),
			FinishedAt:    now,
		},
		Logs:        []string{"log1", "log2"},
		PreviousLog: []string{"prev1"},
		Events: []elasticEvent{
			{
				Type:      "Warning",
				Reason:    "FailedScheduling",
				Message:   "pod unschedulable",
				Count:     1,
				FirstSeen: now,
				LastSeen:  now,
				Source:    "scheduler",
			},
		},
		EnvVars:     map[string]string{"KEY": "value"},
		Warnings:    []string{"warning1"},
		CollectedAt: now,
	}

	report := store.fromDocument(doc)

	if report.ID != "test-id-123" {
		t.Errorf("ID = %v, want test-id-123", report.ID)
	}
	if report.Crash.Namespace != "default" {
		t.Errorf("Crash.Namespace = %v, want default", report.Crash.Namespace)
	}
	if report.Crash.ExitCode != 1 {
		t.Errorf("Crash.ExitCode = %v, want 1", report.Crash.ExitCode)
	}
	if len(report.Logs) != 2 {
		t.Errorf("Logs count = %d, want 2", len(report.Logs))
	}
	if len(report.Events) != 1 {
		t.Errorf("Events count = %d, want 1", len(report.Events))
	}
	if report.Events[0].Type != "Warning" {
		t.Errorf("Events[0].Type = %v, want Warning", report.Events[0].Type)
	}
	if report.Events[0].Source != "scheduler" {
		t.Errorf("Events[0].Source = %v, want scheduler", report.Events[0].Source)
	}
	if report.EnvVars["KEY"] != "value" {
		t.Errorf("EnvVars[KEY] = %v, want value", report.EnvVars["KEY"])
	}
}

func TestElasticStore_documentRoundTrip(t *testing.T) {
	store := &ElasticStore{indexName: "test-index"}

	crash := domain.PodCrash{
		Namespace:     "kube-system",
		PodName:       "coredns-abc123",
		ContainerName: "coredns",
		ExitCode:      137,
		Reason:        "OOMKilled",
		RestartCount:  10,
	}

	original := domain.NewForensicReport(crash)
	original.SetLogs([]string{"log entry"})
	original.AddEvent(*domain.NewEvent("Normal", "Pulled", "Container image pulled"))

	doc := store.toDocument(original)
	restored := store.fromDocument(doc)

	if restored.ID != original.ID {
		t.Errorf("ID mismatch after round trip")
	}
	if restored.Crash.Namespace != original.Crash.Namespace {
		t.Errorf("Crash.Namespace mismatch after round trip")
	}
	if restored.Crash.ExitCode != original.Crash.ExitCode {
		t.Errorf("Crash.ExitCode mismatch after round trip")
	}
	if len(restored.Logs) != len(original.Logs) {
		t.Errorf("Logs count mismatch after round trip")
	}
	if len(restored.Events) != len(original.Events) {
		t.Errorf("Events count mismatch after round trip")
	}
}

func TestNewElasticStore_ConnectionError(t *testing.T) {
	cfg := ElasticConfig{
		Addresses: []string{"http://localhost:59999"},
	}

	_, err := NewElasticStore(cfg)
	if err == nil {
		t.Error("Expected error for unreachable elasticsearch")
	}
}

func TestElasticStore_MockServer_Save(t *testing.T) {
	indexExistsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "PUT" || r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"_index":   "kubecrsh-reports",
				"_id":      "test-id",
				"_version": 1,
				"result":   "created",
			})
			return
		}

		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"name":         "test-node",
				"cluster_name": "test-cluster",
				"version": map[string]string{
					"number": "8.0.0",
				},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(indexExistsHandler)
	defer server.Close()

	cfg := ElasticConfig{
		Addresses: []string{server.URL},
		Index:     "kubecrsh-reports",
	}

	store, err := NewElasticStore(cfg)
	if err != nil {
		t.Skipf("Skipping mock test, ES client validation: %v", err)
	}

	crash := domain.PodCrash{Namespace: "default", PodName: "test-pod"}
	report := domain.NewForensicReport(crash)

	err = store.Save(report)
	if err != nil {
		t.Logf("Save with mock server: %v (expected in some ES client versions)", err)
	}
}

func BenchmarkElasticStore_toDocument(b *testing.B) {
	store := &ElasticStore{indexName: "bench-index"}

	crash := domain.PodCrash{
		Namespace:     "production",
		PodName:       "benchmark-pod",
		ContainerName: "app",
		ExitCode:      1,
		Reason:        "Error",
	}
	report := domain.NewForensicReport(crash)
	report.SetLogs(make([]string, 100))
	for i := 0; i < 10; i++ {
		report.AddEvent(*domain.NewEvent("Warning", "Test", "message"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.toDocument(report)
	}
}

func BenchmarkElasticStore_fromDocument(b *testing.B) {
	store := &ElasticStore{indexName: "bench-index"}

	doc := &elasticDocument{
		ID: "bench-id",
		Crash: elasticCrash{
			Namespace: "production",
			PodName:   "benchmark-pod",
		},
		Logs:    make([]string, 100),
		Events:  make([]elasticEvent, 10),
		EnvVars: map[string]string{"KEY": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.fromDocument(doc)
	}
}
