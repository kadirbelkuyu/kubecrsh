package tui

import (
	"strings"
	"testing"
)

func TestStyles_Initialization(t *testing.T) {
	styles := []struct {
		name  string
		style interface{}
	}{
		{"titleStyle", titleStyle},
		{"subtitleStyle", subtitleStyle},
		{"errorStyle", errorStyle},
		{"warningStyle", warningStyle},
		{"successStyle", successStyle},
		{"infoStyle", infoStyle},
		{"dimStyle", dimStyle},
		{"borderStyle", borderStyle},
		{"helpStyle", helpStyle},
		{"selectedItemStyle", selectedItemStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			if s.style == nil {
				t.Errorf("%s should not be nil", s.name)
			}
		})
	}
}

func TestTitleStyle_Render(t *testing.T) {
	rendered := titleStyle.Render("Test Title")

	if rendered == "" {
		t.Error("Rendered output should not be empty")
	}
	if !strings.Contains(rendered, "Test Title") {
		t.Error("Rendered output should contain the input text")
	}
}

func TestErrorStyle_Render(t *testing.T) {
	rendered := errorStyle.Render("Error Message")

	if rendered == "" {
		t.Error("Rendered output should not be empty")
	}
}

func TestWarningStyle_Render(t *testing.T) {
	rendered := warningStyle.Render("Warning Message")

	if rendered == "" {
		t.Error("Rendered output should not be empty")
	}
}

func TestSuccessStyle_Render(t *testing.T) {
	rendered := successStyle.Render("Success Message")

	if rendered == "" {
		t.Error("Rendered output should not be empty")
	}
}

func TestInfoStyle_Render(t *testing.T) {
	rendered := infoStyle.Render("Info Message")

	if rendered == "" {
		t.Error("Rendered output should not be empty")
	}
}

func TestBorderStyle_Render(t *testing.T) {
	rendered := borderStyle.Render("Content with border")

	if rendered == "" {
		t.Error("Rendered output should not be empty")
	}
}

func TestStylesConsistency(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple text", "Hello"},
		{"with newlines", "Line1\nLine2"},
		{"with special chars", "Status: ✓"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = titleStyle.Render(tc.input)
			_ = errorStyle.Render(tc.input)
			_ = warningStyle.Render(tc.input)
		})
	}
}

func BenchmarkTitleStyle_Render(b *testing.B) {
	text := "Benchmark Title"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		titleStyle.Render(text)
	}
}

func BenchmarkBorderStyle_Render(b *testing.B) {
	text := "Content with multiple lines\nLine 2\nLine 3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		borderStyle.Render(text)
	}
}
