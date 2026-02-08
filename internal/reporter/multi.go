package reporter

import (
	"fmt"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

var _ Storage = (*MultiStore)(nil)

type MultiStore struct {
	primary   Storage
	secondary Storage
}

func NewMultiStore(primary, secondary Storage) *MultiStore {
	return &MultiStore{
		primary:   primary,
		secondary: secondary,
	}
}

func (m *MultiStore) Save(report *domain.ForensicReport) error {
	primaryErr := m.primary.Save(report)
	secondaryErr := m.secondary.Save(report)

	if primaryErr != nil && secondaryErr != nil {
		return fmt.Errorf("both stores failed: primary: %w, secondary: %v", primaryErr, secondaryErr)
	}

	if primaryErr != nil {
		return fmt.Errorf("primary store failed (secondary succeeded): %w", primaryErr)
	}

	if secondaryErr != nil {
		fmt.Printf("Warning: secondary store failed: %v\n", secondaryErr)
	}

	return nil
}

func (m *MultiStore) Load(id string) (*domain.ForensicReport, error) {
	report, err := m.primary.Load(id)
	if err == nil {
		return report, nil
	}

	return m.secondary.Load(id)
}

func (m *MultiStore) List() ([]*domain.ForensicReport, error) {
	return m.primary.List()
}
