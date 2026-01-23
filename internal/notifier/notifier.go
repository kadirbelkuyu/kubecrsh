package notifier

import "github.com/kadirbelkuyu/kubecrsh/internal/domain"

type Notifier interface {
	Notify(report domain.ForensicReport) error
	Name() string
}
