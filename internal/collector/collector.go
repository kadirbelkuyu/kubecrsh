package collector

import (
	"context"
	"fmt"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"k8s.io/client-go/kubernetes"
)

type Collector struct {
	logCollector   *LogCollector
	eventCollector *EventCollector
	envCollector   *EnvCollector
}

func New(client kubernetes.Interface) *Collector {
	return &Collector{
		logCollector:   NewLogCollector(client, 1000),
		eventCollector: NewEventCollector(client),
		envCollector:   NewEnvCollector(client),
	}
}

func (c *Collector) CollectForensics(ctx context.Context, crash domain.PodCrash) (*domain.ForensicReport, error) {
	report := domain.NewForensicReport(crash)

	logs, err := c.logCollector.GetLogs(ctx, crash.Namespace, crash.PodName, crash.ContainerName)
	if err == nil {
		report.SetLogs(logs)
	} else {
		report.AddWarning(fmt.Sprintf("logs: %v", err))
	}

	previousLogs, err := c.logCollector.GetPreviousLogs(ctx, crash.Namespace, crash.PodName, crash.ContainerName)
	if err == nil {
		report.SetPreviousLogs(previousLogs)
	} else {
		report.AddWarning(fmt.Sprintf("previous logs: %v", err))
	}

	events, err := c.eventCollector.GetPodEvents(ctx, crash.Namespace, crash.PodName)
	if err == nil {
		for _, e := range events {
			report.AddEvent(e)
		}
	} else {
		report.AddWarning(fmt.Sprintf("events: %v", err))
	}

	envVars, err := c.envCollector.GetEnvVars(ctx, crash.Namespace, crash.PodName, crash.ContainerName)
	if err == nil {
		for k, v := range envVars {
			report.SetEnvVar(k, v)
		}
	} else {
		report.AddWarning(fmt.Sprintf("env: %v", err))
	}

	return report, nil
}
