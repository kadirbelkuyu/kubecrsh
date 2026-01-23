package views

import (
	"fmt"
	"strings"

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
	dv.updateContent()
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
	(&v).updateContent()
	return v
}

func (v DetailView) SetActiveTab(tab int) DetailView {
	v.ActiveTab = tab
	(&v).updateContent()
	return v
}

func (v DetailView) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Width(v.width).
		Render(fmt.Sprintf("Crash Report: %s", v.report.Summary()))

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Render(fmt.Sprintf("ID: %s | Collected: %s",
			v.report.ID,
			v.report.CollectedAt.Format("2006-01-02 15:04:05"),
		))

	return lipgloss.JoinVertical(lipgloss.Left, title, info)
}

func (v DetailView) renderTabs() string {
	tabs := []string{"Overview", "Logs", "Previous Logs", "Events"}
	var renderedTabs []string

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

func (v DetailView) updateContent() {
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

	b.WriteString(fmt.Sprintf("Namespace:     %s\n", v.report.Crash.Namespace))
	b.WriteString(fmt.Sprintf("Pod:           %s\n", v.report.Crash.PodName))
	b.WriteString(fmt.Sprintf("Container:     %s\n", v.report.Crash.ContainerName))
	b.WriteString(fmt.Sprintf("Reason:        %s\n", v.report.Crash.Reason))
	b.WriteString(fmt.Sprintf("Exit Code:     %d\n", v.report.Crash.ExitCode))
	b.WriteString(fmt.Sprintf("Restart Count: %d\n", v.report.Crash.RestartCount))

	if !v.report.Crash.StartedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Started:       %s\n", v.report.Crash.StartedAt.Format("2006-01-02 15:04:05")))
	}
	if !v.report.Crash.FinishedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Finished:      %s\n", v.report.Crash.FinishedAt.Format("2006-01-02 15:04:05")))
	}

	if len(v.report.EnvVars) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Environment Variables"))
		b.WriteString("\n\n")
		for k, v := range v.report.EnvVars {
			b.WriteString(fmt.Sprintf("%s=%s\n", k, v))
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

	var result string
	for _, line := range logs {
		result += line + "\n"
	}
	return result
}

func (v DetailView) renderEvents() string {
	if len(v.report.Events) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("No events recorded")
	}

	var b strings.Builder
	for _, e := range v.report.Events {
		style := lipgloss.NewStyle()
		if e.IsWarning() {
			style = style.Foreground(lipgloss.Color("#FFB86C"))
		} else {
			style = style.Foreground(lipgloss.Color("#50FA7B"))
		}

		b.WriteString(style.Render(fmt.Sprintf("[%s] %s", e.Type, e.Reason)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s\n", e.Message))
		b.WriteString(fmt.Sprintf("  Count: %d | Last: %s\n\n",
			e.Count,
			e.LastSeen.Format("2006-01-02 15:04:05"),
		))
	}

	return b.String()
}
