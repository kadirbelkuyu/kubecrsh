package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientConfig struct {
	Kubeconfig string
	Context    string
}

func NewClient(cfg ClientConfig) (kubernetes.Interface, error) {
	config, err := buildConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

func buildConfig(cfg ClientConfig) (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	kubeconfig := cfg.Kubeconfig
	if kubeconfig == "" {
		kubeconfig = defaultKubeconfigPath()
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig

	configOverrides := &clientcmd.ConfigOverrides{}
	if cfg.Context != "" {
		configOverrides.CurrentContext = cfg.Context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	return clientConfig.ClientConfig()
}

func defaultKubeconfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
