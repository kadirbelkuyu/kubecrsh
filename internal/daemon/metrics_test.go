package daemon

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()

	if metrics == nil {
		t.Fatal("NewMetrics() returned nil")
	}
	if metrics.CrashesTotal == nil {
		t.Error("CrashesTotal should not be nil")
	}
	if metrics.ReportSize == nil {
		t.Error("ReportSize should not be nil")
	}
	if metrics.NotificationsSent == nil {
		t.Error("NotificationsSent should not be nil")
	}
}

func TestMetrics_CrashesTotal(t *testing.T) {
	metrics := NewMetrics()

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.CrashesTotal)

	metrics.CrashesTotal.WithLabelValues("default", "OOMKilled").Inc()
	metrics.CrashesTotal.WithLabelValues("production", "Error").Add(5)

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "kubecrsh_crashes_total" {
			found = true
			if len(mf.GetMetric()) != 2 {
				t.Errorf("Expected 2 metric series, got %d", len(mf.GetMetric()))
			}
		}
	}
	if !found {
		t.Error("kubecrsh_crashes_total metric not found")
	}
}

func TestMetrics_ReportSize(t *testing.T) {
	metrics := NewMetrics()

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.ReportSize)

	metrics.ReportSize.Observe(1024)
	metrics.ReportSize.Observe(2048)
	metrics.ReportSize.Observe(4096)

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "kubecrsh_report_size_bytes" {
			found = true
			histogram := mf.GetMetric()[0].GetHistogram()
			if histogram.GetSampleCount() != 3 {
				t.Errorf("Expected 3 samples, got %d", histogram.GetSampleCount())
			}
		}
	}
	if !found {
		t.Error("kubecrsh_report_size_bytes metric not found")
	}
}

func TestMetrics_NotificationsSent(t *testing.T) {
	metrics := NewMetrics()

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.NotificationsSent)

	metrics.NotificationsSent.WithLabelValues("slack", "success").Inc()
	metrics.NotificationsSent.WithLabelValues("webhook", "failure").Inc()
	metrics.NotificationsSent.WithLabelValues("slack", "success").Inc()

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "kubecrsh_notifications_sent_total" {
			found = true
			if len(mf.GetMetric()) != 2 {
				t.Errorf("Expected 2 metric series, got %d", len(mf.GetMetric()))
			}
		}
	}
	if !found {
		t.Error("kubecrsh_notifications_sent_total metric not found")
	}
}

func TestMetrics_LabelConsistency(t *testing.T) {
	metrics := NewMetrics()

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.CrashesTotal)

	metrics.CrashesTotal.WithLabelValues("ns1", "reason1").Inc()
	metrics.CrashesTotal.WithLabelValues("ns2", "reason2").Inc()
	metrics.CrashesTotal.WithLabelValues("ns1", "reason1").Inc()

	metricFamilies, _ := registry.Gather()
	for _, mf := range metricFamilies {
		if mf.GetName() == "kubecrsh_crashes_total" {
			for _, m := range mf.GetMetric() {
				if len(m.GetLabel()) != 2 {
					t.Errorf("Expected 2 labels, got %d", len(m.GetLabel()))
				}
			}
		}
	}
}

func BenchmarkMetrics_CrashesTotal(b *testing.B) {
	metrics := NewMetrics()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.CrashesTotal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.CrashesTotal.WithLabelValues("default", "Error").Inc()
	}
}

func BenchmarkMetrics_ReportSize(b *testing.B) {
	metrics := NewMetrics()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.ReportSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.ReportSize.Observe(float64(i * 1024))
	}
}
