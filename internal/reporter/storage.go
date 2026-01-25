package reporter

import "github.com/kadirbelkuyu/kubecrsh/internal/domain"

type Storage interface {
	Save(report *domain.ForensicReport) error
	Load(id string) (*domain.ForensicReport, error)
	List() ([]*domain.ForensicReport, error)
}
