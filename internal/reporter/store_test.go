package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if store.baseDir != tmpDir {
		t.Errorf("baseDir = %v, want %v", store.baseDir, tmpDir)
	}
}

func TestNewStore_EmptyPath(t *testing.T) {
	originalDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	store, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if store.baseDir != "reports" {
		t.Errorf("baseDir = %v, want reports", store.baseDir)
	}
}

func TestNewStore_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newPath := filepath.Join(tmpDir, "nested", "path", "reports")

	_, err := NewStore(newPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("Directory was not created: %s", newPath)
	}
}

func TestStore_Save(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	crash := domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "OOMKilled",
	}
	report := domain.NewForensicReport(crash)
	report.SetLogs([]string{"log line 1", "log line 2"})

	err := store.Save(report)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.json"))
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	data, _ := os.ReadFile(files[0])
	var loaded domain.ForensicReport
	json.Unmarshal(data, &loaded)

	if loaded.ID != report.ID {
		t.Errorf("Loaded ID = %v, want %v", loaded.ID, report.ID)
	}
	if len(loaded.Logs) != 2 {
		t.Errorf("Loaded Logs count = %d, want 2", len(loaded.Logs))
	}
}

func TestStore_Load(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	crash := domain.PodCrash{
		Namespace: "production",
		PodName:   "api-server",
		Reason:    "Error",
		ExitCode:  1,
	}
	report := domain.NewForensicReport(crash)
	store.Save(report)

	loaded, err := store.Load(report.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != report.ID {
		t.Errorf("ID = %v, want %v", loaded.ID, report.ID)
	}
	if loaded.Crash.Namespace != "production" {
		t.Errorf("Crash.Namespace = %v, want production", loaded.Crash.Namespace)
	}
	if loaded.Crash.ExitCode != 1 {
		t.Errorf("Crash.ExitCode = %v, want 1", loaded.Crash.ExitCode)
	}
}

func TestStore_Load_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	_, err := store.Load("nonexistent-id")
	if err == nil {
		t.Error("Expected error for non-existent report")
	}
}

func TestStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	for i := 0; i < 3; i++ {
		crash := domain.PodCrash{
			Namespace: "default",
			PodName:   "pod-" + string(rune('a'+i)),
		}
		report := domain.NewForensicReport(crash)
		store.Save(report)
	}

	reports, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(reports) != 3 {
		t.Errorf("List() returned %d reports, want 3", len(reports))
	}
}

func TestStore_List_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	reports, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(reports) != 0 {
		t.Errorf("List() returned %d reports, want 0", len(reports))
	}
}

func TestStore_List_SkipsInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	crash := domain.PodCrash{Namespace: "default", PodName: "valid-pod"}
	report := domain.NewForensicReport(crash)
	store.Save(report)

	os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not valid json"), 0644)

	reports, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(reports) != 1 {
		t.Errorf("List() returned %d reports, want 1 (should skip invalid)", len(reports))
	}
}

func TestStore_ImplementsStorageInterface(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewStore(tmpDir)

	var _ Storage = store
}

func BenchmarkStore_Save(b *testing.B) {
	tmpDir := b.TempDir()
	store, _ := NewStore(tmpDir)

	crash := domain.PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "Error",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report := domain.NewForensicReport(crash)
		store.Save(report)
	}
}

func BenchmarkStore_List(b *testing.B) {
	tmpDir := b.TempDir()
	store, _ := NewStore(tmpDir)

	for i := 0; i < 100; i++ {
		crash := domain.PodCrash{Namespace: "default", PodName: "pod"}
		report := domain.NewForensicReport(crash)
		store.Save(report)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List()
	}
}
