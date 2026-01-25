package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

var _ Storage = (*Store)(nil)

type Store struct {
	baseDir string
}

func NewStore(baseDir string) (*Store, error) {
	if baseDir == "" {
		baseDir = "reports"
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reports directory: %w", err)
	}

	return &Store{baseDir: baseDir}, nil
}

func (s *Store) Save(report *domain.ForensicReport) error {
	filename := fmt.Sprintf("%s_%s_%s.json",
		report.ID,
		report.Crash.Namespace,
		report.Crash.PodName,
	)

	path := filepath.Join(s.baseDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

func (s *Store) Load(id string) (*domain.ForensicReport, error) {
	files, err := filepath.Glob(filepath.Join(s.baseDir, id+"_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to search for report: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("report not found: %s", id)
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read report: %w", err)
	}

	var report domain.ForensicReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

func (s *Store) List() ([]*domain.ForensicReport, error) {
	files, err := filepath.Glob(filepath.Join(s.baseDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	reports := make([]*domain.ForensicReport, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var report domain.ForensicReport
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}

		reports = append(reports, &report)
	}

	return reports, nil
}
