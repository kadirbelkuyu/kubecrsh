package collector

import (
	"bytes"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type LogCollector struct {
	client    kubernetes.Interface
	tailLines int64
}

func NewLogCollector(client kubernetes.Interface, tailLines int64) *LogCollector {
	if tailLines <= 0 {
		tailLines = 1000
	}
	return &LogCollector{
		client:    client,
		tailLines: tailLines,
	}
}

func (c *LogCollector) GetLogs(ctx context.Context, namespace, podName, containerName string) ([]string, error) {
	return c.getLogs(ctx, namespace, podName, containerName, false)
}

func (c *LogCollector) GetPreviousLogs(ctx context.Context, namespace, podName, containerName string) ([]string, error) {
	return c.getLogs(ctx, namespace, podName, containerName, true)
}

func (c *LogCollector) getLogs(ctx context.Context, namespace, podName, containerName string, previous bool) ([]string, error) {
	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Previous:   previous,
		TailLines:  &c.tailLines,
		Timestamps: true,
	}

	req := c.client.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get log stream: %w", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, stream); err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	logContent := buf.String()
	if logContent == "" {
		return []string{}, nil
	}

	lines := make([]string, 0)
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		if len(line) > 0 {
			lines = append(lines, string(line))
		}
	}

	return lines, nil
}
