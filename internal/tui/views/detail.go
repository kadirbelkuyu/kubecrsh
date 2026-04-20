package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type DetailView struct {
	report    *domain.ForensicReport
	viewport  viewport.Model
	ActiveTab int
	width     int
	height    int
}

func NewDetailView(report *domain.ForensicReport) DetailView {
	vp := viewport.New(80, 20)
	dv := DetailView{
		report:    report,
		viewport:  vp,
		ActiveTab: 0,
	}
	dv.updateContentInternal()
	return dv
}

func (v DetailView) Init() tea.Cmd {
	return nil
}

func (v DetailView) Update(msg tea.Msg) (DetailView, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

func (v DetailView) View() string {
	if v.report == nil {
		return "No report selected"
	}

	header := v.renderHeader()
	tabs := v.renderTabs()
	content := v.renderContent()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabs,
		content,
	)
}

func (v DetailView) SetSize(width, height int) DetailView {
	v.width = width
	v.height = height
	v.viewport.Width = width - 4
	v.viewport.Height = height - 8
	v.updateContentInternal()
	return v
}

func (v DetailView) SetActiveTab(tab int) DetailView {
	v.ActiveTab = tab
	v.updateContentInternal()
	return v
}

func (v DetailView) renderHeader() string {
	if v.width <= 0 {
		return ""
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Width(v.width).
		Render(fmt.Sprintf("Crash Report: %s/%s", v.report.Crash.Namespace, v.report.Crash.PodName))

	meta := lipgloss.JoinHorizontal(
		lipgloss.Left,
		headerTag(v.report.Crash.Reason, lipgloss.Color("#FFB86C"), lipgloss.Color("#1F2937")),
		" ",
		headerTag(fmt.Sprintf("exit:%d", v.report.Crash.ExitCode), lipgloss.Color("#94A3B8"), lipgloss.Color("#111827")),
		" ",
		headerTag(fmt.Sprintf("restart:%d", v.report.Crash.RestartCount), lipgloss.Color("#94A3B8"), lipgloss.Color("#111827")),
		" ",
		headerTag("collected:"+formatTime(v.report.CollectedAt), lipgloss.Color("#8BE9FD"), lipgloss.Color("#0F172A")),
		" ",
		headerTag("age:"+formatSince(v.report.CollectedAt), lipgloss.Color("#50FA7B"), lipgloss.Color("#052E16")),
	)

	timeline := lipgloss.JoinHorizontal(
		lipgloss.Left,
		headerTag("started:"+formatTime(v.report.Crash.StartedAt), lipgloss.Color("#334155"), lipgloss.Color("#E2E8F0")),
		" ",
		headerTag("finished:"+formatTime(v.report.Crash.FinishedAt), lipgloss.Color("#334155"), lipgloss.Color("#E2E8F0")),
		" ",
		headerTag("duration:"+formatCrashDuration(v.report.Crash.StartedAt, v.report.Crash.FinishedAt), lipgloss.Color("#334155"), lipgloss.Color("#E2E8F0")),
		" ",
		headerTag("id:"+shortID(v.report.ID), lipgloss.Color("#64748B"), lipgloss.Color("#F8FAFC")),
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, meta, timeline)
}

func (v DetailView) renderTabs() string {
	tabs := []string{
		"Overview",
		fmt.Sprintf("Logs (%d)", len(v.report.Logs)),
		fmt.Sprintf("Prev Logs (%d)", len(v.report.PreviousLog)),
		fmt.Sprintf("Events (%d)", len(v.report.Events)),
	}
	renderedTabs := make([]string, 0, len(tabs))

	for i, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == v.ActiveTab {
			style = style.
				Foreground(lipgloss.Color("#FF79C6")).
				Bold(true).
				Underline(true)
		} else {
			style = style.Foreground(lipgloss.Color("#6272A4"))
		}
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func (v DetailView) renderContent() string {
	return v.viewport.View()
}

func (v *DetailView) updateContentInternal() {
	var content string

	switch v.ActiveTab {
	case 0:
		content = v.renderOverview()
	case 1:
		content = v.renderLogs(v.report.Logs)
	case 2:
		content = v.renderLogs(v.report.PreviousLog)
	case 3:
		content = v.renderEvents()
	}

	v.viewport.SetContent(content)
}

func (v DetailView) renderOverview() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Crash Details"))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "Namespace:     %s\n", v.report.Crash.Namespace)
	fmt.Fprintf(&b, "Pod:           %s\n", v.report.Crash.PodName)
	fmt.Fprintf(&b, "Container:     %s\n", v.report.Crash.ContainerName)
	fmt.Fprintf(&b, "Reason:        %s\n", v.report.Crash.Reason)
	fmt.Fprintf(&b, "Exit Code:     %d\n", v.report.Crash.ExitCode)
	fmt.Fprintf(&b, "Restart Count: %d\n", v.report.Crash.RestartCount)
	fmt.Fprintf(&b, "Collected At:  %s\n", formatTime(v.report.CollectedAt))
	fmt.Fprintf(&b, "Age:           %s\n", formatSince(v.report.CollectedAt))

	if !v.report.Crash.StartedAt.IsZero() {
		fmt.Fprintf(&b, "Started:       %s\n", formatTime(v.report.Crash.StartedAt))
	}
	if !v.report.Crash.FinishedAt.IsZero() {
		fmt.Fprintf(&b, "Finished:      %s\n", formatTime(v.report.Crash.FinishedAt))
	}
	fmt.Fprintf(&b, "Duration:      %s\n", formatCrashDuration(v.report.Crash.StartedAt, v.report.Crash.FinishedAt))

	if len(v.report.EnvVars) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Environment Variables"))
		b.WriteString("\n\n")
		keys := make([]string, 0, len(v.report.EnvVars))
		for k := range v.report.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&b, "%s=%s\n", key, v.report.EnvVars[key])
		}
	}

	if len(v.report.Warnings) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Warnings"))
		b.WriteString("\n\n")
		for _, warning := range v.report.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}

	return b.String()
}

func (v DetailView) renderLogs(logs []string) string {
	if len(logs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("No logs available")
	}

	var b strings.Builder
	for i, line := range logs {
		fmt.Fprintf(&b, "%5d | %s\n", i+1, line)
	}
	return b.String()
}

func (v DetailView) renderEvents() string {
	if len(v.report.Events) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("No events recorded")
	}

	var b strings.Builder
	events := make([]domain.Event, len(v.report.Events))
	copy(events, v.report.Events)
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].LastSeen.After(events[j].LastSeen)
	})

	for _, e := range events {
		style := lipgloss.NewStyle()
		if e.IsWarning() {
			style = style.Foreground(lipgloss.Color("#FFB86C"))
		} else {
			style = style.Foreground(lipgloss.Color("#50FA7B"))
		}

		b.WriteString(style.Render(fmt.Sprintf("[%s] [%s] %s",
			formatTimeOnly(e.LastSeen),
			e.Type,
			e.Reason,
		)))
		b.WriteString("\n")
		fmt.Fprintf(&b, "  %s\n", e.Message)
		fmt.Fprintf(&b, "  Count: %d | First: %s | Last: %s\n\n",
			e.Count,
			formatTime(e.FirstSeen),
			formatTime(e.LastSeen),
		)
	}

	return b.String()
}

func headerTag(label string, bg lipgloss.Color, fg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Render(label)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func formatTimeOnly(t time.Time) string {
	if t.IsZero() {
		return "--:--:--"
	}
	return t.Local().Format("15:04:05")
}

func formatSince(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	d := time.Since(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func formatCrashDuration(start, finish time.Time) string {
	if start.IsZero() || finish.IsZero() {
		return "unknown"
	}

	d := finish.Sub(start)
	if d < 0 {
		d = 0
	}
	return d.Round(time.Second).String()
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
