package reporter

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

var _ Storage = (*Store)(nil)
var _ SaveWithResult = (*Store)(nil)

type Store struct {
	baseDir     string
	compression string
	mu          sync.RWMutex
}

type Option func(*Store)

func WithCompression(compression string) Option {
	return func(s *Store) {
		s.compression = strings.TrimSpace(strings.ToLower(compression))
	}
}

func NewStore(baseDir string, opts ...Option) (*Store, error) {
	if baseDir == "" {
		baseDir = "reports"
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reports directory: %w", err)
	}

	s := &Store{baseDir: baseDir, compression: "none"}
	for _, opt := range opts {
		opt(s)
	}

	if s.compression == "" {
		s.compression = "none"
	}

	return s, nil
}

func (s *Store) Save(report *domain.ForensicReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.saveLocked(report)
	return err
}

func (s *Store) SaveWithResult(report *domain.ForensicReport) (SaveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(report)
}

func (s *Store) saveLocked(report *domain.ForensicReport) (SaveResult, error) {

	ext := ".json"
	if s.compression == "gzip" || s.compression == "gz" {
		ext = ".json.gz"
	}

	filename := fmt.Sprintf("%s_%s_%s%s",
		report.ID,
		report.Crash.Namespace,
		report.Crash.PodName,
		ext,
	)

	path := filepath.Join(s.baseDir, filename)

	tmp, err := os.CreateTemp(s.baseDir, report.ID+"_*.tmp")
	if err != nil {
		return SaveResult{}, fmt.Errorf("failed to create temp report: %w", err)
	}
	tmpPath := tmp.Name()
	var bytesWritten int64

	writeErr := func() error {
		defer tmp.Close()

		var w io.Writer = tmp
		if ext == ".json.gz" {
			gw := gzip.NewWriter(tmp)
			defer gw.Close()
			w = gw
		}
		w = &countingWriter{w: w, n: &bytesWritten}

		enc := json.NewEncoder(w)
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("failed to encode report: %w", err)
		}

		if err := tmp.Sync(); err != nil {
			return fmt.Errorf("failed to sync report: %w", err)
		}

		return nil
	}()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return SaveResult{}, writeErr
	}

	if err := replaceFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return SaveResult{}, fmt.Errorf("failed to move report into place: %w", err)
	}

	return SaveResult{BytesWritten: bytesWritten, Path: path}, nil
}

func (s *Store) Load(id string) (*domain.ForensicReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := s.globByID(id)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("report not found: %s", id)
	}

	var report domain.ForensicReport
	if err := readJSONFile(files[0], &report); err != nil {
		return nil, err
	}

	return &report, nil
}

func (s *Store) List() ([]*domain.ForensicReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := s.globAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	reports := make([]*domain.ForensicReport, 0, len(files))
	for _, file := range files {
		var report domain.ForensicReport
		if err := readJSONFile(file, &report); err != nil {
			continue
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

func (s *Store) globByID(id string) ([]string, error) {
	jsonFiles, err := filepath.Glob(filepath.Join(s.baseDir, id+"_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to search for report: %w", err)
	}
	gzFiles, err := filepath.Glob(filepath.Join(s.baseDir, id+"_*.json.gz"))
	if err != nil {
		return nil, fmt.Errorf("failed to search for report: %w", err)
	}
	return append(jsonFiles, gzFiles...), nil
}

func (s *Store) globAll() ([]string, error) {
	jsonFiles, err := filepath.Glob(filepath.Join(s.baseDir, "*.json"))
	if err != nil {
		return nil, err
	}
	gzFiles, err := filepath.Glob(filepath.Join(s.baseDir, "*.json.gz"))
	if err != nil {
		return nil, err
	}
	return append(jsonFiles, gzFiles...), nil
}

func readJSONFile(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open report: %w", err)
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("failed to open gzip report: %w", err)
		}
		defer gr.Close()
		r = gr
	}

	dec := json.NewDecoder(r)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("failed to decode report: %w", err)
	}

	return nil
}

type countingWriter struct {
	w io.Writer
	n *int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	*c.n += int64(n)
	return n, err
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.Rename(src, dst)
}
