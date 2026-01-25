package views

import (
	"strings"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func createTestReport(namespace, podName, reason string, exitCode int32) *domain.ForensicReport {
	crash := domain.PodCrash{
		Namespace:    namespace,
		PodName:      podName,
		Reason:       reason,
		ExitCode:     exitCode,
		RestartCount: 3,
	}
	return domain.NewForensicReport(crash)
}

func TestCrashItem_Title(t *testing.T) {
	report := createTestReport("production", "api-server", "OOMKilled", 137)
	item := crashItem{report: *report}

	title := item.Title()

	if !strings.Contains(title, "production") {
		t.Errorf("Title should contain namespace, got: %s", title)
	}
	if !strings.Contains(title, "api-server") {
		t.Errorf("Title should contain pod name, got: %s", title)
	}
}

func TestCrashItem_Description(t *testing.T) {
	report := createTestReport("default", "test-pod", "Error", 1)
	item := crashItem{report: *report}

	desc := item.Description()

	if !strings.Contains(desc, "Error") {
		t.Errorf("Description should contain reason, got: %s", desc)
	}
	if !strings.Contains(desc, "1") {
		t.Errorf("Description should contain exit code, got: %s", desc)
	}
	if !strings.Contains(desc, "3") {
		t.Errorf("Description should contain restart count, got: %s", desc)
	}
}

func TestCrashItem_FilterValue(t *testing.T) {
	report := createTestReport("default", "unique-pod-name", "Error", 1)
	item := crashItem{report: *report}

	filterValue := item.FilterValue()

	if filterValue != "unique-pod-name" {
		t.Errorf("FilterValue = %v, want unique-pod-name", filterValue)
	}
}

func TestNewListView(t *testing.T) {
	reports := []*domain.ForensicReport{
		createTestReport("ns1", "pod1", "OOMKilled", 137),
		createTestReport("ns2", "pod2", "Error", 1),
	}

	view := NewListView(reports)

	if view.list.Title != "Crash Reports" {
		t.Errorf("List title = %v, want 'Crash Reports'", view.list.Title)
	}
}

func TestNewListView_Empty(t *testing.T) {
	view := NewListView([]*domain.ForensicReport{})

	if !view.IsEmpty() {
		t.Error("IsEmpty() should return true for empty list")
	}
}

func TestListView_Init(t *testing.T) {
	view := NewListView(nil)

	cmd := view.Init()

	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestListView_View(t *testing.T) {
	reports := []*domain.ForensicReport{
		createTestReport("default", "test-pod", "Error", 1),
	}
	view := NewListView(reports)

	output := view.View()

	if output == "" {
		t.Error("View() should not return empty string")
	}
}

func TestListView_SetSize(t *testing.T) {
	view := NewListView(nil)

	view = view.SetSize(80, 24)

	if view.width != 80 {
		t.Errorf("width = %d, want 80", view.width)
	}
	if view.height != 24 {
		t.Errorf("height = %d, want 24", view.height)
	}
}

func TestListView_IsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		reports []*domain.ForensicReport
		want    bool
	}{
		{"empty list", []*domain.ForensicReport{}, true},
		{"nil list", nil, true},
		{"with reports", []*domain.ForensicReport{createTestReport("ns", "pod", "Error", 1)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewListView(tt.reports)
			if got := view.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListView_EmptyMessage(t *testing.T) {
	view := NewListView(nil)

	msg := view.EmptyMessage()

	if msg == "" {
		t.Error("EmptyMessage() should not return empty string")
	}
	if !strings.Contains(msg, "No crash reports") {
		t.Errorf("EmptyMessage should contain 'No crash reports', got: %s", msg)
	}
}

func TestListView_SelectedReport_NoSelection(t *testing.T) {
	view := NewListView(nil)

	selected := view.SelectedReport()

	if selected != nil {
		t.Error("SelectedReport() should return nil for empty list")
	}
}

func TestListView_AddReport(t *testing.T) {
	view := NewListView(nil)

	if !view.IsEmpty() {
		t.Error("Initial view should be empty")
	}

	report := domain.ForensicReport{
		ID: "test-id",
		Crash: domain.PodCrash{
			Namespace: "default",
			PodName:   "added-pod",
		},
		CollectedAt: time.Now(),
	}

	view = view.AddReport(report)

	if view.IsEmpty() {
		t.Error("View should not be empty after adding report")
	}
}

func BenchmarkNewListView(b *testing.B) {
	reports := make([]*domain.ForensicReport, 100)
	for i := 0; i < 100; i++ {
		reports[i] = createTestReport("ns", "pod", "Error", 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewListView(reports)
	}
}

func BenchmarkCrashItem_Title(b *testing.B) {
	report := createTestReport("production", "api-server", "OOMKilled", 137)
	item := crashItem{report: *report}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Title()
	}
}
