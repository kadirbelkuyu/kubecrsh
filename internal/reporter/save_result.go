package reporter

import "github.com/kadirbelkuyu/kubecrsh/internal/domain"

type SaveResult struct {
	BytesWritten int64
	Path         string
}

type SaveWithResult interface {
	SaveWithResult(report *domain.ForensicReport) (SaveResult, error)
}

