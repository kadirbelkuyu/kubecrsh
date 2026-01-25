package domain

import (
	"testing"
	"time"
)

func TestPodCrash_IsOOMKilled(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   bool
	}{
		{"OOMKilled returns true", "OOMKilled", true},
		{"Error returns false", "Error", false},
		{"CrashLoopBackOff returns false", "CrashLoopBackOff", false},
		{"empty returns false", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PodCrash{Reason: tt.reason}
			if got := p.IsOOMKilled(); got != tt.want {
				t.Errorf("IsOOMKilled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodCrash_IsCrashLoopBackOff(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   bool
	}{
		{"CrashLoopBackOff returns true", "CrashLoopBackOff", true},
		{"Error returns false", "Error", false},
		{"OOMKilled returns false", "OOMKilled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PodCrash{Reason: tt.reason}
			if got := p.IsCrashLoopBackOff(); got != tt.want {
				t.Errorf("IsCrashLoopBackOff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodCrash_FullName(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		podName   string
		want      string
	}{
		{"standard case", "default", "nginx-pod", "default/nginx-pod"},
		{"with dashes", "kube-system", "coredns-abc123", "kube-system/coredns-abc123"},
		{"empty namespace", "", "pod", "/pod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PodCrash{Namespace: tt.namespace, PodName: tt.podName}
			if got := p.FullName(); got != tt.want {
				t.Errorf("FullName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPodCrash(t *testing.T) {
	crash := NewPodCrash("default", "test-pod", "main")

	if crash.Namespace != "default" {
		t.Errorf("Namespace = %v, want default", crash.Namespace)
	}
	if crash.PodName != "test-pod" {
		t.Errorf("PodName = %v, want test-pod", crash.PodName)
	}
	if crash.ContainerName != "main" {
		t.Errorf("ContainerName = %v, want main", crash.ContainerName)
	}
}

func TestPodCrash_WithExitCode(t *testing.T) {
	crash := &PodCrash{
		Namespace:    "production",
		PodName:      "api-server",
		ExitCode:     137,
		Reason:       "OOMKilled",
		RestartCount: 5,
		StartedAt:    time.Now().Add(-1 * time.Hour),
		FinishedAt:   time.Now(),
	}

	if crash.ExitCode != 137 {
		t.Errorf("ExitCode = %v, want 137", crash.ExitCode)
	}
	if !crash.IsOOMKilled() {
		t.Error("Expected OOMKilled to be true")
	}
	if crash.RestartCount != 5 {
		t.Errorf("RestartCount = %v, want 5", crash.RestartCount)
	}
}
