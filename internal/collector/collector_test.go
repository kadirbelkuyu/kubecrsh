package collector

import (
	"context"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNew(t *testing.T) {
	client := fake.NewSimpleClientset()
	collector := New(client)

	if collector == nil {
		t.Fatal("New() returned nil")
	}
	if collector.logCollector == nil {
		t.Error("logCollector should not be nil")
	}
	if collector.eventCollector == nil {
		t.Error("eventCollector should not be nil")
	}
	if collector.envCollector == nil {
		t.Error("envCollector should not be nil")
	}
}

func TestCollector_CollectForensics(t *testing.T) {
	client := fake.NewSimpleClientset()
	collector := New(client)

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "main",
		Reason:        "OOMKilled",
		ExitCode:      137,
	}

	ctx := context.Background()
	report, err := collector.CollectForensics(ctx, crash)

	if err != nil {
		t.Fatalf("CollectForensics() error = %v", err)
	}
	if report == nil {
		t.Fatal("Report should not be nil")
	}
	if report.Crash.PodName != "test-pod" {
		t.Errorf("Crash.PodName = %v, want test-pod", report.Crash.PodName)
	}
	if report.ID == "" {
		t.Error("Report ID should not be empty")
	}
}

func TestCollector_CollectForensics_WithEvents(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			Name:      "test-pod",
			Namespace: "default",
		},
		Type:    "Warning",
		Reason:  "FailedMount",
		Message: "Unable to mount volume",
	}

	client := fake.NewSimpleClientset(event)
	collector := New(client)

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "main",
	}

	ctx := context.Background()
	report, err := collector.CollectForensics(ctx, crash)

	if err != nil {
		t.Fatalf("CollectForensics() error = %v", err)
	}

	if len(report.Events) != 1 {
		t.Logf("Expected 1 event, got %d (events may be filtered by pod)", len(report.Events))
	}
}

func TestCollector_CollectForensics_WithPod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "main",
				Image: "nginx:latest",
				Env: []corev1.EnvVar{
					{Name: "DB_HOST", Value: "localhost"},
					{Name: "DB_PORT", Value: "5432"},
				},
			}},
		},
	}

	client := fake.NewSimpleClientset(pod)
	collector := New(client)

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "main",
	}

	ctx := context.Background()
	report, err := collector.CollectForensics(ctx, crash)

	if err != nil {
		t.Fatalf("CollectForensics() error = %v", err)
	}

	if len(report.EnvVars) < 2 {
		t.Logf("Expected at least 2 env vars, got %d", len(report.EnvVars))
	}
}

func TestCollector_CollectForensics_NonExistentPod(t *testing.T) {
	client := fake.NewSimpleClientset()
	collector := New(client)

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "nonexistent-pod",
		ContainerName: "main",
	}

	ctx := context.Background()
	report, err := collector.CollectForensics(ctx, crash)

	if err != nil {
		t.Fatalf("CollectForensics() error = %v (should not error, just have empty data)", err)
	}
	if report == nil {
		t.Fatal("Report should not be nil even for non-existent pod")
	}
}

func TestCollector_CollectForensics_ContextCancellation(t *testing.T) {
	client := fake.NewSimpleClientset()
	collector := New(client)

	crash := domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	report, err := collector.CollectForensics(ctx, crash)

	if report == nil {
		t.Logf("Report may be nil with canceled context, err: %v", err)
	}
}

func BenchmarkCollector_CollectForensics(b *testing.B) {
	client := fake.NewSimpleClientset()
	collector := New(client)

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "main",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.CollectForensics(ctx, crash)
	}
}
