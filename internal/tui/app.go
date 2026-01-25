package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kadirbelkuyu/kubecrsh/internal/collector"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/internal/reporter"
	"github.com/kadirbelkuyu/kubecrsh/internal/tui/views"
	"k8s.io/client-go/kubernetes"
)

type viewState int

const (
	stateList viewState = iota
	stateDetail
)

type model struct {
	state      viewState
	listView   views.ListView
	detailView views.DetailView
	help       help.Model
	width      int
	height     int
	client     kubernetes.Interface
	collector  *collector.Collector
	store      *reporter.Store
	err        error
}

type crashMsg struct {
	crash domain.PodCrash
}

type reportMsg struct {
	report domain.ForensicReport
}

type reportsLoadedMsg struct {
	reports []*domain.ForensicReport
}

type errMsg struct {
	err error
}

func New(client kubernetes.Interface, store *reporter.Store) model {
	return model{
		state:     stateList,
		listView:  views.NewListView([]*domain.ForensicReport{}),
		help:      help.New(),
		client:    client,
		collector: collector.New(client),
		store:     store,
	}
}

func (m model) Init() tea.Cmd {
	return m.loadReports()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.listView = m.listView.SetSize(msg.Width, msg.Height-2)
		if m.state == stateDetail {
			m.detailView = m.detailView.SetSize(msg.Width, msg.Height-2)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc", "backspace":
			if m.state == stateDetail {
				m.state = stateList
				return m, nil
			}

		case "enter":
			if m.state == stateList {
				if report := m.listView.SelectedReport(); report != nil {
					m.detailView = views.NewDetailView(report)
					m.detailView = m.detailView.SetSize(m.width, m.height-2)
					m.state = stateDetail
					return m, nil
				}
			}

		case "tab":
			if m.state == stateDetail {
				activeTab := (m.detailView.ActiveTab + 1) % 4
				m.detailView = m.detailView.SetActiveTab(activeTab)
				return m, nil
			}
		}

	case crashMsg:
		return m, m.collectForensics(msg.crash)

	case reportMsg:
		if err := m.store.Save(&msg.report); err != nil {
			m.err = err
		}
		m.listView = m.listView.AddReport(msg.report)
		return m, nil

	case reportsLoadedMsg:
		m.listView = views.NewListView(msg.reports)
		m.listView = m.listView.SetSize(m.width, m.height-2)
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	switch m.state {
	case stateList:
		m.listView, cmd = m.listView.Update(msg)
	case stateDetail:
		m.detailView, cmd = m.detailView.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var view string
	switch m.state {
	case stateList:
		if m.listView.IsEmpty() {
			view = m.listView.EmptyMessage()
		} else {
			view = m.listView.View()
		}
	case stateDetail:
		view = m.detailView.View()
	}

	help := helpStyle.Render(m.help.ShortHelpView(keys.ShortHelp()))

	if m.err != nil {
		errMsg := errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		return fmt.Sprintf("%s\n\n%s\n%s", view, errMsg, help)
	}

	return fmt.Sprintf("%s\n%s", view, help)
}

func (m model) loadReports() tea.Cmd {
	return func() tea.Msg {
		reports, err := m.store.List()
		if err != nil {
			return errMsg{err}
		}
		return reportsLoadedMsg{reports: reports}
	}
}

func (m model) collectForensics(crash domain.PodCrash) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		report, err := m.collector.CollectForensics(ctx, crash)
		if err != nil {
			return errMsg{err}
		}
		return reportMsg{report: *report}
	}
}

func OnCrash(crash domain.PodCrash) tea.Msg {
	return crashMsg{crash: crash}
}
