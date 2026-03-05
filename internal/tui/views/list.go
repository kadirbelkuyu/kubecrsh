package views

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

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
	return fmt.Sprintf("%s (exit: %d) - %d restarts - collected %s",
		i.report.Crash.Reason,
		i.report.Crash.ExitCode,
		i.report.Crash.RestartCount,
		formatDateTime(i.report.CollectedAt),
	)
}

func (i crashItem) FilterValue() string {
	search := strings.Join([]string{
		i.report.Crash.Namespace,
		i.report.Crash.PodName,
		i.report.Crash.ContainerName,
		i.report.Crash.Reason,
		i.report.ID,
		formatDateTime(i.report.CollectedAt),
		fmt.Sprintf("%d", i.report.Crash.ExitCode),
		fmt.Sprintf("%d", i.report.Crash.RestartCount),
	}, " ")

	parts := []string{
		"ns=" + i.report.Crash.Namespace,
		"pod=" + i.report.Crash.PodName,
		"container=" + i.report.Crash.ContainerName,
		"reason=" + i.report.Crash.Reason,
		"id=" + i.report.ID,
		fmt.Sprintf("exit=%d", i.report.Crash.ExitCode),
		fmt.Sprintf("restart=%d", i.report.Crash.RestartCount),
		fmt.Sprintf("warn=%d", len(i.report.Warnings)),
		"time=" + formatDateTime(i.report.CollectedAt),
		"search=" + search,
	}

	return strings.ToLower(strings.Join(parts, "|"))
}

type crashDelegate struct{}

func newCrashDelegate() crashDelegate {
	return crashDelegate{}
}

func (d crashDelegate) Height() int {
	return 2
}

func (d crashDelegate) Spacing() int {
	return 1
}

func (d crashDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d crashDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(crashItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	prefix := "  "
	if isSelected {
		prefix = "> "
	}

	pod := fmt.Sprintf("%s/%s", ci.report.Crash.Namespace, ci.report.Crash.PodName)
	line1 := prefix + lipgloss.JoinHorizontal(
		lipgloss.Left,
		itemTitleStyle(isSelected).Render(pod),
		" ",
		reasonTag(ci.report.Crash.Reason, isSelected),
		" ",
		neutralTag(fmt.Sprintf("exit:%d", ci.report.Crash.ExitCode), isSelected),
		" ",
		neutralTag(fmt.Sprintf("restart:%d", ci.report.Crash.RestartCount), isSelected),
	)

	reportID := ci.report.ID
	if len(reportID) > 8 {
		reportID = reportID[:8]
	}
	line2 := "  " + lipgloss.JoinHorizontal(
		lipgloss.Left,
		timeTag(formatDateTime(ci.report.CollectedAt), isSelected),
		" ",
		ageTag("age:"+formatAge(ci.report.CollectedAt), isSelected),
		" ",
		idTag("id:"+reportID, isSelected),
	)

	if len(ci.report.Warnings) > 0 {
		line2 += " " + warnTag(fmt.Sprintf("warn:%d", len(ci.report.Warnings)), isSelected)
	}

	_, _ = fmt.Fprintf(w, "%s\n%s", line1, line2)
}

type ListView struct {
	list   list.Model
	width  int
	height int
}

func NewListView(reports []*domain.ForensicReport) ListView {
	items := itemsFromReports(sortReports(reports))
	delegate := newCrashDelegate()

	l := list.New(items, delegate, 0, 0)
	l.Title = "Crash Watch - / search, c clear | fields: ns: reason: pod: container: id: exit: restart: warn:"
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)
	l.Filter = crashFilter
	l.FilterInput.Prompt = "Search/Filter> "
	l.FilterInput.Placeholder = "error ns:prod reason:oom exit:137 warn:yes"
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("report", "reports")

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

func (v ListView) ResetFilter() ListView {
	v.list.ResetFilter()
	return v
}

func (v ListView) SettingFilter() bool {
	return v.list.SettingFilter()
}

func (v ListView) IsEmpty() bool {
	return len(v.list.Items()) == 0
}

func (v ListView) EmptyMessage() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Render("No crash reports yet. Waiting for pod crashes...")
}

func itemsFromReports(reports []*domain.ForensicReport) []list.Item {
	items := make([]list.Item, 0, len(reports))
	for _, r := range reports {
		if r == nil {
			continue
		}
		items = append(items, crashItem{report: *r})
	}
	return items
}

func sortReports(reports []*domain.ForensicReport) []*domain.ForensicReport {
	copied := make([]*domain.ForensicReport, 0, len(reports))
	for _, r := range reports {
		if r == nil {
			continue
		}
		copied = append(copied, r)
	}

	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].CollectedAt.After(copied[j].CollectedAt)
	})

	return copied
}

func formatDateTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func formatAge(t time.Time) string {
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

func itemTitleStyle(selected bool) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true)
	if selected {
		return base.Foreground(lipgloss.Color("#FF79C6"))
	}
	return base.Foreground(lipgloss.Color("#F8F8F2"))
}

func reasonTag(reason string, selected bool) string {
	color := lipgloss.Color("#FFB86C")
	switch reason {
	case "OOMKilled":
		color = lipgloss.Color("#FF5555")
	case "CrashLoopBackOff":
		color = lipgloss.Color("#F1FA8C")
	case "Error":
		color = lipgloss.Color("#FF6B6B")
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1E1F29")).
		Background(color).
		Padding(0, 1)
	if selected {
		style = style.Bold(true)
	}
	return style.Render(reason)
}

func neutralTag(label string, selected bool) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E6E6E6")).
		Background(lipgloss.Color("#3A3D4A")).
		Padding(0, 1)
	if selected {
		style = style.Foreground(lipgloss.Color("#FFFFFF"))
	}
	return style.Render(label)
}

func timeTag(label string, selected bool) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0F172A")).
		Background(lipgloss.Color("#8BE9FD")).
		Padding(0, 1)
	if selected {
		style = style.Bold(true)
	}
	return style.Render(label)
}

func ageTag(label string, selected bool) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0B1020")).
		Background(lipgloss.Color("#50FA7B")).
		Padding(0, 1)
	if selected {
		style = style.Bold(true)
	}
	return style.Render(label)
}

func idTag(label string, selected bool) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		Background(lipgloss.Color("#6272A4")).
		Padding(0, 1)
	if selected {
		style = style.Bold(true)
	}
	return style.Render(label)
}

func warnTag(label string, selected bool) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1A1A1A")).
		Background(lipgloss.Color("#FFB86C")).
		Padding(0, 1)
	if selected {
		style = style.Bold(true)
	}
	return style.Render(label)
}

func crashFilter(term string, targets []string) []list.Rank {
	trimmed := strings.TrimSpace(strings.ToLower(term))
	if trimmed == "" {
		out := make([]list.Rank, 0, len(targets))
		for i := range targets {
			out = append(out, list.Rank{Index: i})
		}
		return out
	}

	fieldTerms, freeTerms := parseQueryTerms(trimmed)
	ranks := make([]list.Rank, 0, len(targets))
	for i, target := range targets {
		fields := parseTargetFields(target)
		if matchesQuery(fields, fieldTerms, freeTerms) {
			ranks = append(ranks, list.Rank{Index: i})
		}
	}

	return ranks
}

func parseTargetFields(target string) map[string]string {
	fields := make(map[string]string, 10)
	for _, part := range strings.Split(target, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		fields[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return fields
}

func parseQueryTerms(q string) (map[string]string, []string) {
	fieldTerms := make(map[string]string)
	freeTerms := make([]string, 0)

	for _, token := range strings.Fields(q) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		parts := strings.SplitN(token, ":", 2)
		if len(parts) != 2 {
			freeTerms = append(freeTerms, token)
			continue
		}

		key := normalizeField(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			freeTerms = append(freeTerms, token)
			continue
		}

		fieldTerms[key] = val
	}

	return fieldTerms, freeTerms
}

func normalizeField(field string) string {
	switch strings.TrimSpace(field) {
	case "ns", "namespace":
		return "ns"
	case "pod":
		return "pod"
	case "container", "c":
		return "container"
	case "reason":
		return "reason"
	case "id":
		return "id"
	case "exit", "exitcode":
		return "exit"
	case "restart", "restarts":
		return "restart"
	case "warn", "warning", "warnings":
		return "warn"
	case "time":
		return "time"
	default:
		return ""
	}
}

func matchesQuery(fields map[string]string, fieldTerms map[string]string, freeTerms []string) bool {
	for key, val := range fieldTerms {
		if !matchField(fields, key, val) {
			return false
		}
	}

	searchBase := fields["search"]
	for _, token := range freeTerms {
		if !strings.Contains(searchBase, token) {
			return false
		}
	}

	return true
}

func matchField(fields map[string]string, key, value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return true
	}

	if key == "warn" {
		warnCount, _ := strconv.Atoi(fields["warn"])
		switch value {
		case "yes", "true", "1", "gt0", "has":
			return warnCount > 0
		case "no", "false", "0", "none":
			return warnCount == 0
		}
	}

	fieldValue := strings.ToLower(fields[key])
	if fieldValue == "" {
		return false
	}

	if key == "exit" || key == "restart" || key == "warn" {
		if _, err := strconv.Atoi(value); err == nil {
			return fieldValue == value
		}
	}

	return strings.Contains(fieldValue, value)
}
