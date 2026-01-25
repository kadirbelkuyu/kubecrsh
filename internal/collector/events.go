package collector

import (
	"context"
	"fmt"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type EventCollector struct {
	client kubernetes.Interface
}

func NewEventCollector(client kubernetes.Interface) *EventCollector {
	return &EventCollector{client: client}
}

func (c *EventCollector) GetPodEvents(ctx context.Context, namespace, podName string) ([]domain.Event, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", podName)

	events, err := c.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	result := make([]domain.Event, 0, len(events.Items))
	for _, e := range events.Items {
		result = append(result, domain.Event{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Count:     e.Count,
			FirstSeen: e.FirstTimestamp.Time,
			LastSeen:  e.LastTimestamp.Time,
			Source:    e.Source.Component,
		})
	}

	return result, nil
}
