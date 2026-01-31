package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Kubeconfig string
	Context    string
	Namespace  string
	Reports    ReportsConfig
	API        APIConfig
	Watch      WatchConfig
}

type ReportsConfig struct {
	Path        string          `mapstructure:"path"`
	Retention   time.Duration   `mapstructure:"retention"`
	Compression string          `mapstructure:"compression"`
	Redaction   RedactionConfig `mapstructure:"redaction"`
}

type RedactionConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	EnvAllowlist     []string `mapstructure:"env_allowlist"`
	EnvDenylist      []string `mapstructure:"env_denylist"`
	LogPatterns      []string `mapstructure:"log_patterns"`
	Replacement      string   `mapstructure:"replacement"`
	RedactFromSource bool     `mapstructure:"redact_from_source"`
}

type APIConfig struct {
	ReportsEnabled bool   `mapstructure:"reports_enabled"`
	Token          string `mapstructure:"token"`
	AllowFull      bool   `mapstructure:"allow_full"`
}

type WatchConfig struct {
	Reasons []string
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		v.AddConfigPath(filepath.Join(home, ".kubecrsh"))
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	v.SetDefault("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	v.SetDefault("context", "")
	v.SetDefault("namespace", "")
	v.SetDefault("reports.path", "reports")
	v.SetDefault("reports.retention", "168h")
	v.SetDefault("reports.compression", "none")
	v.SetDefault("reports.redaction.enabled", false)
	v.SetDefault("reports.redaction.env_allowlist", []string{})
	v.SetDefault("reports.redaction.env_denylist", []string{})
	v.SetDefault("reports.redaction.log_patterns", []string{})
	v.SetDefault("reports.redaction.replacement", "***")
	v.SetDefault("reports.redaction.redact_from_source", false)
	v.SetDefault("api.reports_enabled", false)
	v.SetDefault("api.token", "")
	v.SetDefault("api.allow_full", false)
	v.SetDefault("watch.reasons", []string{"OOMKilled", "Error", "CrashLoopBackOff"})

	v.AutomaticEnv()
	v.SetEnvPrefix("KUBECRSH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
