package views

import (
	"strings"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

func TestFormatTime_Zero(t *testing.T) {
	if got := formatTime(time.Time{}); got != "unknown" {
		t.Errorf("formatTime(zero) = %q, want unknown", got)
	}
}

func TestFormatCrashDuration(t *testing.T) {
	start := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	end := start.Add(42 * time.Second)

	if got := formatCrashDuration(start, end); got != "42s" {
		t.Errorf("formatCrashDuration() = %q, want 42s", got)
	}
}

func TestDetailView_RenderTabs_Counts(t *testing.T) {
	report := domain.NewForensicReport(domain.PodCrash{
		Namespace: "default",
		PodName:   "api",
		Reason:    "Error",
	})
	report.Logs = []string{"l1", "l2"}
	report.PreviousLog = []string{"p1"}
	report.Events = []domain.Event{{Type: "Warning", Reason: "BackOff", Message: "msg"}}

	view := NewDetailView(report)
	tabs := view.renderTabs()

	if !strings.Contains(tabs, "Logs (2)") {
		t.Errorf("tabs should contain Logs (2), got: %s", tabs)
	}
	if !strings.Contains(tabs, "Prev Logs (1)") {
		t.Errorf("tabs should contain Prev Logs (1), got: %s", tabs)
	}
	if !strings.Contains(tabs, "Events (1)") {
		t.Errorf("tabs should contain Events (1), got: %s", tabs)
	}
}
