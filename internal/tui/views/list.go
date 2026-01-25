package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type crashItem struct {
	report domain.ForensicReport
}

func (i crashItem) Title() string {
	return fmt.Sprintf("%s/%s", i.report.Crash.Namespace, i.report.Crash.PodName)
}

func (i crashItem) Description() string {
	return fmt.Sprintf("%s (exit: %d) - %d restarts",
		i.report.Crash.Reason,
		i.report.Crash.ExitCode,
		i.report.Crash.RestartCount,
	)
}

func (i crashItem) FilterValue() string {
	return i.report.Crash.PodName
}

type ListView struct {
	list   list.Model
	width  int
	height int
}

func NewListView(reports []*domain.ForensicReport) ListView {
	items := make([]list.Item, len(reports))
	for i, r := range reports {
		items[i] = crashItem{report: *r}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF79C6")).
		Bold(true)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BD93F9"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Crash Reports"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return ListView{list: l}
}

func (v ListView) Init() tea.Cmd {
	return nil
}

func (v ListView) Update(msg tea.Msg) (ListView, tea.Cmd) {
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v ListView) View() string {
	return v.list.View()
}

func (v ListView) SetSize(width, height int) ListView {
	v.width = width
	v.height = height
	v.list.SetSize(width, height)
	return v
}

func (v ListView) SelectedReport() *domain.ForensicReport {
	if item, ok := v.list.SelectedItem().(crashItem); ok {
		return &item.report
	}
	return nil
}

func (v ListView) AddReport(report domain.ForensicReport) ListView {
	v.list.InsertItem(0, crashItem{report: report})
	return v
}

func (v ListView) IsEmpty() bool {
	return len(v.list.Items()) == 0
}

func (v ListView) EmptyMessage() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Render("No crash reports yet. Watching for pod crashes...")
}
