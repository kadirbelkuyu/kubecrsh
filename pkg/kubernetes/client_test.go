package kubernetes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientConfig(t *testing.T) {
	cfg := ClientConfig{
		Kubeconfig: "/path/to/kubeconfig",
		Context:    "production",
	}

	if cfg.Kubeconfig != "/path/to/kubeconfig" {
		t.Errorf("Kubeconfig = %v, want /path/to/kubeconfig", cfg.Kubeconfig)
	}
	if cfg.Context != "production" {
		t.Errorf("Context = %v, want production", cfg.Context)
	}
}

func TestDefaultKubeconfigPath(t *testing.T) {
	path := defaultKubeconfigPath()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	expected := filepath.Join(home, ".kube", "config")
	if path != expected {
		t.Errorf("defaultKubeconfigPath() = %v, want %v", path, expected)
	}
}

func TestNewClient_InvalidKubeconfig(t *testing.T) {
	cfg := ClientConfig{
		Kubeconfig: "/nonexistent/path/kubeconfig",
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Error("Expected error for invalid kubeconfig path")
	}
}

func TestNewClient_EmptyConfig(t *testing.T) {
	cfg := ClientConfig{}

	_, err := NewClient(cfg)

	if err != nil {
		t.Logf("NewClient() with empty config returned: %v (expected if no valid kubeconfig)", err)
	}
}

func TestBuildConfig_WithContext(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	kubeconfigContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
- context:
    cluster: test-cluster
    user: test-user
  name: production-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}

	cfg := ClientConfig{
		Kubeconfig: kubeconfigPath,
		Context:    "production-context",
	}

	_, err := buildConfig(cfg)
	if err != nil {
		t.Logf("buildConfig() error = %v", err)
	}
}

func TestNewClient_WithValidKubeconfig(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	kubeconfigContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}

	cfg := ClientConfig{
		Kubeconfig: kubeconfigPath,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Error("Expected non-nil client")
	}
}

func TestClientConfig_EmptyValues(t *testing.T) {
	cfg := ClientConfig{}

	if cfg.Kubeconfig != "" {
		t.Errorf("Kubeconfig should be empty by default")
	}
	if cfg.Context != "" {
		t.Errorf("Context should be empty by default")
	}
}

func BenchmarkDefaultKubeconfigPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		defaultKubeconfigPath()
	}
}
