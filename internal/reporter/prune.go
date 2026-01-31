package reporter

import (
	"fmt"
	"os"
	"time"
)

type PruneResult struct {
	Deleted int
	Kept    int
	Failed  int
}

func (s *Store) Prune(retention time.Duration) (PruneResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var res PruneResult
	if retention <= 0 {
		return res, nil
	}

	files, err := s.globAll()
	if err != nil {
		return res, fmt.Errorf("failed to list reports: %w", err)
	}

	now := time.Now()
	var firstErr error

	for _, path := range files {
		collectedAt, err := readCollectedAt(path)
		if err != nil {
			info, statErr := os.Stat(path)
			if statErr != nil {
				res.Failed++
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to stat report: %w", statErr)
				}
				continue
			}
			collectedAt = info.ModTime()
		}

		if now.Sub(collectedAt) <= retention {
			res.Kept++
			continue
		}

		if err := os.Remove(path); err != nil {
			res.Failed++
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to delete report: %w", err)
			}
			continue
		}

		res.Deleted++
	}

	return res, firstErr
}

func readCollectedAt(path string) (time.Time, error) {
	var v struct {
		CollectedAt time.Time `json:"CollectedAt"`
	}

	if err := readJSONFile(path, &v); err != nil {
		return time.Time{}, err
	}

	if v.CollectedAt.IsZero() {
		return time.Time{}, fmt.Errorf("missing CollectedAt")
	}

	return v.CollectedAt, nil
}
