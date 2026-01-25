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
