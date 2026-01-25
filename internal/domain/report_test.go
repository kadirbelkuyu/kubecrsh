package domain

import (
	"strings"
	"testing"
)

func TestNewForensicReport(t *testing.T) {
	crash := PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "OOMKilled",
	}

	report := NewForensicReport(crash)

	if report.ID == "" {
		t.Error("ID should not be empty")
	}
	if len(report.ID) != 16 {
		t.Errorf("ID length = %d, want 16", len(report.ID))
	}
	if report.Crash.Namespace != "default" {
		t.Errorf("Crash.Namespace = %v, want default", report.Crash.Namespace)
	}
	if report.EnvVars == nil {
		t.Error("EnvVars should be initialized")
	}
	if report.Events == nil {
		t.Error("Events should be initialized")
	}
	if report.CollectedAt.IsZero() {
		t.Error("CollectedAt should not be zero")
	}
}

func TestForensicReport_AddEvent(t *testing.T) {
	report := NewForensicReport(PodCrash{})

	event1 := Event{Type: "Warning", Reason: "FailedMount"}
	event2 := Event{Type: "Normal", Reason: "Scheduled"}

	report.AddEvent(event1)
	report.AddEvent(event2)

	if len(report.Events) != 2 {
		t.Errorf("Events count = %d, want 2", len(report.Events))
	}
	if report.Events[0].Reason != "FailedMount" {
		t.Errorf("First event reason = %v, want FailedMount", report.Events[0].Reason)
	}
}

func TestForensicReport_SetLogs(t *testing.T) {
	report := NewForensicReport(PodCrash{})
	logs := []string{"line1", "line2", "line3"}

	report.SetLogs(logs)

	if len(report.Logs) != 3 {
		t.Errorf("Logs count = %d, want 3", len(report.Logs))
	}
}

func TestForensicReport_SetPreviousLogs(t *testing.T) {
	report := NewForensicReport(PodCrash{})
	logs := []string{"previous line 1", "previous line 2"}

	report.SetPreviousLogs(logs)

	if len(report.PreviousLog) != 2 {
		t.Errorf("PreviousLog count = %d, want 2", len(report.PreviousLog))
	}
}

func TestForensicReport_SetEnvVar(t *testing.T) {
	report := NewForensicReport(PodCrash{})

	report.SetEnvVar("DATABASE_URL", "postgres://localhost")
	report.SetEnvVar("API_KEY", "secret")

	if len(report.EnvVars) != 2 {
		t.Errorf("EnvVars count = %d, want 2", len(report.EnvVars))
	}
	if report.EnvVars["DATABASE_URL"] != "postgres://localhost" {
		t.Errorf("DATABASE_URL = %v, want postgres://localhost", report.EnvVars["DATABASE_URL"])
	}
}

func TestForensicReport_WarningCount(t *testing.T) {
	tests := []struct {
		name   string
		events []Event
		want   int
	}{
		{
			name:   "no events",
			events: []Event{},
			want:   0,
		},
		{
			name: "all warnings",
			events: []Event{
				{Type: "Warning"},
				{Type: "Warning"},
			},
			want: 2,
		},
		{
			name: "mixed events",
			events: []Event{
				{Type: "Warning"},
				{Type: "Normal"},
				{Type: "Warning"},
			},
			want: 2,
		},
		{
			name: "no warnings",
			events: []Event{
				{Type: "Normal"},
				{Type: "Normal"},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewForensicReport(PodCrash{})
			for _, e := range tt.events {
				report.AddEvent(e)
			}

			if got := report.WarningCount(); got != tt.want {
				t.Errorf("WarningCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForensicReport_Summary(t *testing.T) {
	crash := PodCrash{
		Namespace: "production",
		PodName:   "api-server",
		Reason:    "OOMKilled",
	}
	report := NewForensicReport(crash)

	summary := report.Summary()

	if !strings.Contains(summary, "production/api-server") {
		t.Errorf("Summary should contain full name, got: %s", summary)
	}
	if !strings.Contains(summary, "OOMKilled") {
		t.Errorf("Summary should contain reason, got: %s", summary)
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func BenchmarkNewForensicReport(b *testing.B) {
	crash := PodCrash{
		Namespace: "default",
		PodName:   "test-pod",
		Reason:    "Error",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewForensicReport(crash)
	}
}

func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateID()
	}
}
