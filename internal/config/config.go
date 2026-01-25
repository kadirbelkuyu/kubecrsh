package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Kubeconfig string
	Context    string
	Namespace  string
	Reports    ReportsConfig
	Watch      WatchConfig
}

type ReportsConfig struct {
	Path      string
	Retention time.Duration
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
	v.SetDefault("watch.reasons", []string{"OOMKilled", "Error", "CrashLoopBackOff"})

	v.AutomaticEnv()
	v.SetEnvPrefix("KUBECRSH")

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
