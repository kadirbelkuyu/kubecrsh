package domain

import (
	"testing"
)

func TestEvent_IsWarning(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      bool
	}{
		{"Warning type", "Warning", true},
		{"Normal type", "Normal", false},
		{"empty type", "", false},
		{"unknown type", "Unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{Type: tt.eventType}
			if got := e.IsWarning(); got != tt.want {
				t.Errorf("IsWarning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_IsNormal(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      bool
	}{
		{"Normal type", "Normal", true},
		{"Warning type", "Warning", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{Type: tt.eventType}
			if got := e.IsNormal(); got != tt.want {
				t.Errorf("IsNormal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEvent(t *testing.T) {
	event := NewEvent("Warning", "FailedScheduling", "0/3 nodes are available")

	if event.Type != "Warning" {
		t.Errorf("Type = %v, want Warning", event.Type)
	}
	if event.Reason != "FailedScheduling" {
		t.Errorf("Reason = %v, want FailedScheduling", event.Reason)
	}
	if event.Message != "0/3 nodes are available" {
		t.Errorf("Message = %v, want '0/3 nodes are available'", event.Message)
	}
}

func TestEvent_FullLifecycle(t *testing.T) {
	event := NewEvent("Normal", "Scheduled", "Pod scheduled successfully")

	if event.IsWarning() {
		t.Error("Normal event should not be warning")
	}
	if !event.IsNormal() {
		t.Error("Normal event should return true for IsNormal")
	}
}
