package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Reports.Path != "reports" {
		t.Errorf("Reports.Path = %v, want reports", cfg.Reports.Path)
	}
	if cfg.Reports.Retention != 168*time.Hour {
		t.Errorf("Reports.Retention = %v, want 168h", cfg.Reports.Retention)
	}
	if cfg.Performance.CrashWorkers != 4 {
		t.Errorf("Performance.CrashWorkers = %d, want 4", cfg.Performance.CrashWorkers)
	}
	if cfg.Performance.CrashQueueSize != 512 {
		t.Errorf("Performance.CrashQueueSize = %d, want 512", cfg.Performance.CrashQueueSize)
	}
	if cfg.Performance.NotifierWorkers != 4 {
		t.Errorf("Performance.NotifierWorkers = %d, want 4", cfg.Performance.NotifierWorkers)
	}
	if cfg.Performance.NotifierQueueSize != 1024 {
		t.Errorf("Performance.NotifierQueueSize = %d, want 1024", cfg.Performance.NotifierQueueSize)
	}
	if cfg.LeaderElection.Enabled {
		t.Errorf("LeaderElection.Enabled = %v, want false", cfg.LeaderElection.Enabled)
	}
	if cfg.LeaderElection.LeaseName != "kubecrsh-leader-election" {
		t.Errorf("LeaderElection.LeaseName = %q, want kubecrsh-leader-election", cfg.LeaderElection.LeaseName)
	}
	if cfg.LeaderElection.LeaseDuration != 15*time.Second {
		t.Errorf("LeaderElection.LeaseDuration = %v, want 15s", cfg.LeaderElection.LeaseDuration)
	}
	if cfg.LeaderElection.RenewDeadline != 10*time.Second {
		t.Errorf("LeaderElection.RenewDeadline = %v, want 10s", cfg.LeaderElection.RenewDeadline)
	}
	if cfg.LeaderElection.RetryPeriod != 2*time.Second {
		t.Errorf("LeaderElection.RetryPeriod = %v, want 2s", cfg.LeaderElection.RetryPeriod)
	}

	expectedReasons := []string{"OOMKilled", "Error", "CrashLoopBackOff"}
	if len(cfg.Watch.Reasons) != len(expectedReasons) {
		t.Errorf("Watch.Reasons length = %d, want %d", len(cfg.Watch.Reasons), len(expectedReasons))
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
kubeconfig: /custom/kubeconfig
context: production
namespace: monitoring
reports:
  path: /var/reports
  retention: 72h
performance:
  crash_workers: 8
  crash_queue_size: 2048
  notifier_workers: 3
  notifier_queue_size: 4096
leader_election:
  enabled: true
  lease_name: kubecrsh-test-lock
  lease_namespace: kubecrsh
  lease_duration: 30s
  renew_deadline: 20s
  retry_period: 5s
watch:
  reasons:
    - OOMKilled
    - Error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Kubeconfig != "/custom/kubeconfig" {
		t.Errorf("Kubeconfig = %v, want /custom/kubeconfig", cfg.Kubeconfig)
	}
	if cfg.Context != "production" {
		t.Errorf("Context = %v, want production", cfg.Context)
	}
	if cfg.Namespace != "monitoring" {
		t.Errorf("Namespace = %v, want monitoring", cfg.Namespace)
	}
	if cfg.Reports.Path != "/var/reports" {
		t.Errorf("Reports.Path = %v, want /var/reports", cfg.Reports.Path)
	}
	if cfg.Reports.Retention != 72*time.Hour {
		t.Errorf("Reports.Retention = %v, want 72h", cfg.Reports.Retention)
	}
	if cfg.Performance.CrashWorkers != 8 {
		t.Errorf("Performance.CrashWorkers = %d, want 8", cfg.Performance.CrashWorkers)
	}
	if cfg.Performance.CrashQueueSize != 2048 {
		t.Errorf("Performance.CrashQueueSize = %d, want 2048", cfg.Performance.CrashQueueSize)
	}
	if cfg.Performance.NotifierWorkers != 3 {
		t.Errorf("Performance.NotifierWorkers = %d, want 3", cfg.Performance.NotifierWorkers)
	}
	if cfg.Performance.NotifierQueueSize != 4096 {
		t.Errorf("Performance.NotifierQueueSize = %d, want 4096", cfg.Performance.NotifierQueueSize)
	}
	if !cfg.LeaderElection.Enabled {
		t.Errorf("LeaderElection.Enabled = %v, want true", cfg.LeaderElection.Enabled)
	}
	if cfg.LeaderElection.LeaseName != "kubecrsh-test-lock" {
		t.Errorf("LeaderElection.LeaseName = %q, want kubecrsh-test-lock", cfg.LeaderElection.LeaseName)
	}
	if cfg.LeaderElection.LeaseNamespace != "kubecrsh" {
		t.Errorf("LeaderElection.LeaseNamespace = %q, want kubecrsh", cfg.LeaderElection.LeaseNamespace)
	}
	if cfg.LeaderElection.LeaseDuration != 30*time.Second {
		t.Errorf("LeaderElection.LeaseDuration = %v, want 30s", cfg.LeaderElection.LeaseDuration)
	}
	if cfg.LeaderElection.RenewDeadline != 20*time.Second {
		t.Errorf("LeaderElection.RenewDeadline = %v, want 20s", cfg.LeaderElection.RenewDeadline)
	}
	if cfg.LeaderElection.RetryPeriod != 5*time.Second {
		t.Errorf("LeaderElection.RetryPeriod = %v, want 5s", cfg.LeaderElection.RetryPeriod)
	}
	if len(cfg.Watch.Reasons) != 2 {
		t.Errorf("Watch.Reasons length = %d, want 2", len(cfg.Watch.Reasons))
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
kubeconfig: [invalid yaml
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/non/existent/path/config.yaml")
	if err == nil {
		t.Log("Config loaded with defaults for non-existent file")
	}
	if cfg != nil && cfg.Reports.Path == "" {
		t.Error("Expected defaults to be set")
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	original := os.Getenv("KUBECRSH_NAMESPACE")
	defer os.Setenv("KUBECRSH_NAMESPACE", original)

	os.Setenv("KUBECRSH_NAMESPACE", "env-namespace")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Namespace != "env-namespace" {
		t.Errorf("Namespace = %v, want env-namespace", cfg.Namespace)
	}
}

func TestLoad_EnvironmentVariables_NestedKeys(t *testing.T) {
	origEnabled := os.Getenv("KUBECRSH_API_REPORTS_ENABLED")
	origRetention := os.Getenv("KUBECRSH_REPORTS_RETENTION")
	defer os.Setenv("KUBECRSH_API_REPORTS_ENABLED", origEnabled)
	defer os.Setenv("KUBECRSH_REPORTS_RETENTION", origRetention)

	os.Setenv("KUBECRSH_API_REPORTS_ENABLED", "true")
	os.Setenv("KUBECRSH_REPORTS_RETENTION", "72h")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.API.ReportsEnabled != true {
		t.Errorf("API.ReportsEnabled = %v, want true", cfg.API.ReportsEnabled)
	}
	if cfg.Reports.Retention != 72*time.Hour {
		t.Errorf("Reports.Retention = %v, want 72h", cfg.Reports.Retention)
	}
}

func TestConfig_EmptyNamespace(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Namespace != "" {
		t.Logf("Namespace has value: %s (may be from env)", cfg.Namespace)
	}
}

func TestReportsConfig_RetentionDuration(t *testing.T) {
	tests := []struct {
		name      string
		retention string
		want      time.Duration
	}{
		{"1 hour", "1h", 1 * time.Hour},
		{"24 hours", "24h", 24 * time.Hour},
		{"7 days", "168h", 168 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			content := "reports:\n  retention: " + tt.retention
			os.WriteFile(configPath, []byte(content), 0644)

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.Reports.Retention != tt.want {
				t.Errorf("Retention = %v, want %v", cfg.Reports.Retention, tt.want)
			}
		})
	}
}
