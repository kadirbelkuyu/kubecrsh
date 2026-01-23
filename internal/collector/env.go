package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type EnvCollector struct {
	client kubernetes.Interface
}

func NewEnvCollector(client kubernetes.Interface) *EnvCollector {
	return &EnvCollector{client: client}
}

func (c *EnvCollector) GetEnvVars(ctx context.Context, namespace, podName, containerName string) (map[string]string, error) {
	pod, err := c.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	envVars := make(map[string]string)

	for _, container := range pod.Spec.Containers {
		if container.Name != containerName {
			continue
		}

		for _, env := range container.Env {
			if env.Value != "" {
				envVars[env.Name] = env.Value
			} else if env.ValueFrom != nil {
				envVars[env.Name] = c.resolveEnvSource(env)
			}
		}
		break
	}

	return envVars, nil
}

func (c *EnvCollector) resolveEnvSource(env interface{}) string {
	switch v := env.(type) {
	default:
		_ = v
		return "[from-source]"
	}
}
