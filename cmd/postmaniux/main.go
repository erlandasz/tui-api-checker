package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/domain"
	"github.com/erlandas/postmaniux/internal/envmanager"
	"github.com/erlandas/postmaniux/internal/httpclient"
	"github.com/erlandas/postmaniux/internal/storage"
	"github.com/erlandas/postmaniux/internal/tui/reqeditor"
	"github.com/erlandas/postmaniux/internal/tui/respview"
	"github.com/erlandas/postmaniux/internal/tui/tree"
)

// Pane enum for focus management. Cycling through panes with Ctrl+W.
type pane int

const (
	paneTree pane = iota
	paneRequest
	paneResponse
	paneCount
)

type model struct {
	tree        tree.Model
	reqEditor   reqeditor.Model
	respView    respview.Model
	store       *storage.FileStore
	client      *httpclient.Client
	activeEnv   *domain.Environment
	focusedPane pane
	width       int
	height      int
	err         error
}

// responseMsg wraps the async HTTP result so bubbletea can deliver it.
type responseMsg struct {
	resp *httpclient.Response
	err  error
}

func initialModel(store *storage.FileStore) model {
	ctx := context.Background()
	names, _ := store.ListCollections(ctx)
	var collections []domain.Collection
	for _, n := range names {
		col, err := store.LoadCollection(ctx, n)
		if err == nil {
			collections = append(collections, col)
		}
	}

	m := model{
		tree:      tree.New(collections),
		reqEditor: reqeditor.New(),
		respView:  respview.New(),
		store:     store,
		client:    httpclient.NewClient(),
	}
	m.tree.SetFocused(true)
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutPanes()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+w":
			m.cycleFocus()
			return m, nil
		}

	case tree.RequestSelectedMsg:
		m.reqEditor.SetRequest(msg.Request)
		return m, nil

	case reqeditor.SendRequestMsg:
		req := msg.Request
		if m.activeEnv != nil {
			req = envmanager.ResolveRequest(req, *m.activeEnv)
		}
		return m, m.executeRequest(req)

	case responseMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.respView.SetResponse(msg.resp)
			m.err = nil
		}
		return m, nil
	}

	// Delegate to focused child
	var cmd tea.Cmd
	switch m.focusedPane {
	case paneTree:
		m.tree, cmd = m.tree.Update(msg)
	case paneRequest:
		m.reqEditor, cmd = m.reqEditor.Update(msg)
	case paneResponse:
		m.respView, cmd = m.respView.Update(msg)
	}
	return m, cmd
}

func (m *model) cycleFocus() {
	m.tree.SetFocused(false)
	m.reqEditor.SetFocused(false)
	m.respView.SetFocused(false)

	m.focusedPane = (m.focusedPane + 1) % paneCount

	switch m.focusedPane {
	case paneTree:
		m.tree.SetFocused(true)
	case paneRequest:
		m.reqEditor.SetFocused(true)
	case paneResponse:
		m.respView.SetFocused(true)
	}
}

func (m *model) layoutPanes() {
	leftW := m.width * 30 / 100
	rightW := m.width - leftW - 1
	topH := m.height * 40 / 100
	bottomH := m.height - topH - 1

	m.tree.SetSize(leftW, m.height)
	m.reqEditor.SetSize(rightW, topH)
	m.respView.SetSize(rightW, bottomH)
}

func (m model) executeRequest(req domain.Request) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		resp, err := client.Do(context.Background(), req)
		if err != nil {
			return responseMsg{err: err}
		}
		return responseMsg{resp: &resp}
	}
}

func (m model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	leftW := m.width * 30 / 100
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	focusBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86"))

	treeStyle := borderStyle
	reqStyle := borderStyle
	respStyle := borderStyle
	switch m.focusedPane {
	case paneTree:
		treeStyle = focusBorderStyle
	case paneRequest:
		reqStyle = focusBorderStyle
	case paneResponse:
		respStyle = focusBorderStyle
	}

	leftPane := treeStyle.Width(leftW - 2).Height(m.height - 2).Render(m.tree.View())
	rightW := m.width - leftW - 1
	topH := m.height * 40 / 100
	bottomH := m.height - topH - 1

	topPane := reqStyle.Width(rightW - 2).Height(topH - 2).Render(m.reqEditor.View())
	bottomPane := respStyle.Width(rightW - 2).Height(bottomH - 2).Render(m.respView.View())

	// Error bar
	if m.err != nil {
		errLine := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err))
		_ = errLine
	}

	rightSide := lipgloss.JoinVertical(lipgloss.Left, topPane, bottomPane)
	layout := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightSide)

	v := tea.NewView(layout)
	v.AltScreen = true
	return v
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	store := storage.NewFileStore(filepath.Join(home, ".postmaniux"))
	// bubbletea v2 uses alt screen by default
	p := tea.NewProgram(initialModel(store))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
