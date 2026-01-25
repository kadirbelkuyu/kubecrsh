package watcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNew(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}

	watcher := New(client, handler)

	if watcher == nil {
		t.Fatal("New() returned nil")
	}
	if watcher.client == nil {
		t.Error("client should not be nil")
	}
	if watcher.handler == nil {
		t.Error("handler should not be nil")
	}
	if len(watcher.reasons) != 3 {
		t.Errorf("Expected 3 default reasons, got %d", len(watcher.reasons))
	}
}

func TestNew_WithOptions(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}

	watcher := New(client, handler,
		WithNamespace("production"),
		WithReasons([]string{"CustomReason"}),
	)

	if watcher.namespace != "production" {
		t.Errorf("namespace = %v, want production", watcher.namespace)
	}
	if !watcher.reasons["CustomReason"] {
		t.Error("CustomReason should be in reasons map")
	}
}

func TestWithNamespace(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}

	watcher := New(client, handler, WithNamespace("kube-system"))

	if watcher.namespace != "kube-system" {
		t.Errorf("namespace = %v, want kube-system", watcher.namespace)
	}
}

func TestWithReasons(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}

	watcher := New(client, handler, WithReasons([]string{"Reason1", "Reason2"}))

	if !watcher.reasons["Reason1"] {
		t.Error("Reason1 should be in reasons")
	}
	if !watcher.reasons["Reason2"] {
		t.Error("Reason2 should be in reasons")
	}
}

func TestWatcher_shouldHandle(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}

	watcher := New(client, handler)

	tests := []struct {
		reason string
		want   bool
	}{
		{"OOMKilled", true},
		{"Error", true},
		{"CrashLoopBackOff", true},
		{"Unknown", false},
		{"Completed", false},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if got := watcher.shouldHandle(tt.reason); got != tt.want {
				t.Errorf("shouldHandle(%s) = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}

func TestWatcher_createCrashFromTerminated(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}
	watcher := New(client, handler)

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	cs := corev1.ContainerStatus{
		Name: "main",
		State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode:   137,
				Reason:     "OOMKilled",
				Signal:     9,
				StartedAt:  now,
				FinishedAt: now,
			},
		},
		RestartCount: 5,
	}

	crash := watcher.createCrashFromTerminated(pod, cs)

	if crash == nil {
		t.Fatal("Expected crash, got nil")
	}
	if crash.Namespace != "default" {
		t.Errorf("Namespace = %v, want default", crash.Namespace)
	}
	if crash.PodName != "test-pod" {
		t.Errorf("PodName = %v, want test-pod", crash.PodName)
	}
	if crash.ExitCode != 137 {
		t.Errorf("ExitCode = %v, want 137", crash.ExitCode)
	}
	if crash.Reason != "OOMKilled" {
		t.Errorf("Reason = %v, want OOMKilled", crash.Reason)
	}
}

func TestWatcher_createCrashFromTerminated_UnhandledReason(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}
	watcher := New(client, handler)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	cs := corev1.ContainerStatus{
		Name: "main",
		State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
				Reason:   "Completed",
			},
		},
	}

	crash := watcher.createCrashFromTerminated(pod, cs)

	if crash != nil {
		t.Error("Expected nil for unhandled reason")
	}
}

func TestWatcher_createCrashLoopBackOff(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}
	watcher := New(client, handler)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crash-pod",
			Namespace: "production",
		},
	}

	cs := corev1.ContainerStatus{
		Name:         "app",
		RestartCount: 10,
		State: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{
				Reason: "CrashLoopBackOff",
			},
		},
		LastTerminationState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
				Signal:   15,
			},
		},
	}

	crash := watcher.createCrashLoopBackOff(pod, cs)

	if crash == nil {
		t.Fatal("Expected crash, got nil")
	}
	if crash.Reason != "CrashLoopBackOff" {
		t.Errorf("Reason = %v, want CrashLoopBackOff", crash.Reason)
	}
	if crash.RestartCount != 10 {
		t.Errorf("RestartCount = %v, want 10", crash.RestartCount)
	}
	if crash.ExitCode != 1 {
		t.Errorf("ExitCode = %v, want 1", crash.ExitCode)
	}
}

func TestWatcher_detectCrashes(t *testing.T) {
	var mu sync.Mutex
	var crashes []domain.PodCrash

	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {
		mu.Lock()
		crashes = append(crashes, crash)
		mu.Unlock()
	}
	watcher := New(client, handler)

	now := metav1.Now()
	oldPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
	}

	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode:   1,
						Reason:     "Error",
						FinishedAt: now,
					},
				},
			}},
		},
	}

	watcher.detectCrashes(oldPod, newPod)

	mu.Lock()
	defer mu.Unlock()
	if len(crashes) != 1 {
		t.Fatalf("Expected 1 crash, got %d", len(crashes))
	}
	if crashes[0].PodName != "test-pod" {
		t.Errorf("PodName = %v, want test-pod", crashes[0].PodName)
	}
}

func TestWatcher_Start_ContextCancellation(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}
	watcher := New(client, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("Start() returned: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return after context cancellation")
	}
}

func TestWatcher_checkPodOnAdd(t *testing.T) {
	var mu sync.Mutex
	var crashes []domain.PodCrash

	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {
		mu.Lock()
		crashes = append(crashes, crash)
		mu.Unlock()
	}
	watcher := New(client, handler)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crashed-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason: "CrashLoopBackOff",
					},
				},
				RestartCount: 5,
			}},
		},
	}

	watcher.checkPodOnAdd(pod)

	mu.Lock()
	defer mu.Unlock()
	if len(crashes) != 1 {
		t.Fatalf("Expected 1 crash on add, got %d", len(crashes))
	}
}

func TestWatcher_EmptyReasonDefault(t *testing.T) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}
	watcher := New(client, handler)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	cs := corev1.ContainerStatus{
		Name: "main",
		State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
				Reason:   "",
			},
		},
	}

	crash := watcher.createCrashFromTerminated(pod, cs)

	if crash == nil {
		t.Fatal("Expected crash with default reason")
	}
	if crash.Reason != "Error" {
		t.Errorf("Reason = %v, want Error (default)", crash.Reason)
	}
}

func BenchmarkWatcher_detectCrashes(b *testing.B) {
	client := fake.NewSimpleClientset()
	handler := func(crash domain.PodCrash) {}
	watcher := New(client, handler)

	oldPod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
	}

	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "main",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 1,
						Reason:   "Error",
					},
				},
			}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.detectCrashes(oldPod, newPod)
	}
}
