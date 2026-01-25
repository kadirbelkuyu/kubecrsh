//go:build integration

package collector

import (
	"context"
	"os"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/pkg/kubernetes"
)

func TestIntegration_CollectForensics_RealCluster(t *testing.T) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		t.Skip("KUBECONFIG not set, skipping integration test")
	}

	client, err := kubernetes.NewClient(kubernetes.ClientConfig{
		Kubeconfig: kubeconfig,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	collector := New(client.Clientset())

	crash := domain.PodCrash{
		Namespace:     "default",
		PodName:       "test-pod",
		ContainerName: "main",
	}

	ctx := context.Background()
	report, err := collector.CollectForensics(ctx, crash)

	if err != nil {
		t.Logf("CollectForensics() error = %v (expected if pod doesn't exist)", err)
	}
	if report != nil {
		t.Logf("Report collected: ID=%s, Events=%d", report.ID, len(report.Events))
	}
}
