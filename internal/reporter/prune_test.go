package reporter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func TestStore_Prune_DeletesOldReports(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	oldReport := domain.NewForensicReport(domain.PodCrash{Namespace: "default", PodName: "old"})
	oldReport.CollectedAt = time.Now().Add(-2 * time.Hour)
	if err := store.Save(oldReport); err != nil {
		t.Fatalf("Save(old) error = %v", err)
	}

	newReport := domain.NewForensicReport(domain.PodCrash{Namespace: "default", PodName: "new"})
	newReport.CollectedAt = time.Now()
	if err := store.Save(newReport); err != nil {
		t.Fatalf("Save(new) error = %v", err)
	}

	res, err := store.Prune(1 * time.Hour)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}

	if res.Deleted != 1 {
		t.Fatalf("Deleted = %d, want 1", res.Deleted)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.json"))
	if len(files) != 1 {
		t.Fatalf("files = %d, want 1", len(files))
	}

	if _, err := os.Stat(files[0]); err != nil {
		t.Fatalf("remaining file missing: %v", err)
	}
}

func TestStore_GzipCompression_SaveLoadList(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir, WithCompression("gzip"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	report := domain.NewForensicReport(domain.PodCrash{Namespace: "default", PodName: "p"})
	if err := store.Save(report); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.json.gz"))
	if len(files) != 1 {
		t.Fatalf("Expected 1 gz file, got %d", len(files))
	}

	loaded, err := store.Load(report.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ID != report.ID {
		t.Fatalf("loaded.ID = %s, want %s", loaded.ID, report.ID)
	}

	reports, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("List() = %d, want 1", len(reports))
	}
}
