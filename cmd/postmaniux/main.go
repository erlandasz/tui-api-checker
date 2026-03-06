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
	"github.com/erlandas/postmaniux/internal/tui/envpicker"
	"github.com/erlandas/postmaniux/internal/tui/help"
	"github.com/erlandas/postmaniux/internal/tui/newreq"
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
	envPicker   envpicker.Model
	helpOverlay help.Model
	newReq      newreq.Model
	store       *storage.FileStore
	client      *httpclient.Client
	activeEnv   *domain.Environment
	focusedPane pane
	width       int
	height      int
	status      string
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

	// Load environments for picker
	envNames, _ := store.ListEnvironments(ctx)
	var envs []domain.Environment
	for _, n := range envNames {
		env, err := store.LoadEnvironment(ctx, n)
		if err == nil {
			envs = append(envs, env)
		}
	}

	m := model{
		tree:        tree.New(collections),
		reqEditor:   reqeditor.New(),
		respView:    respview.New(),
		envPicker:   envpicker.New(envs),
		helpOverlay: help.New(),
		newReq:      newreq.New(),
		store:       store,
		client:      httpclient.NewClient(),
	}
	m.tree.SetFocused(true)
	m.reqEditor.SetEnvironment(nil) // initialize with date vars only
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
		// When help overlay is visible, delegate all input to it
		if m.helpOverlay.Visible() {
			var cmd tea.Cmd
			m.helpOverlay, cmd = m.helpOverlay.Update(msg)
			return m, cmd
		}

		// When env picker is visible, delegate all input to it
		if m.envPicker.Visible() {
			var cmd tea.Cmd
			m.envPicker, cmd = m.envPicker.Update(msg)
			return m, cmd
		}

		// When new request modal is visible, delegate all input to it
		if m.newReq.Visible() {
			var cmd tea.Cmd
			m.newReq, cmd = m.newReq.Update(msg)
			return m, cmd
		}

		// When reqeditor is editing, delegate all input to it
		if m.focusedPane == paneRequest && m.reqEditor.Editing() {
			var cmd tea.Cmd
			m.reqEditor, cmd = m.reqEditor.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focusedPane != paneRequest {
				return m, tea.Quit
			}
		case "ctrl+w":
			m.cycleFocus()
			return m, nil
		case "ctrl+e":
			m.envPicker.Toggle()
			return m, nil
		case "?":
			if m.focusedPane != paneRequest {
				m.helpOverlay.Toggle()
				return m, nil
			}
		}

	case help.DismissMsg:
		return m, nil

	case envpicker.EnvSelectedMsg:
		env := msg.Env
		m.activeEnv = &env
		m.reqEditor.SetEnvironment(env.Variables)
		m.status = fmt.Sprintf("Environment: %s", env.Name)
		return m, nil

	case envpicker.EnvSavedMsg:
		ctx := context.Background()
		if err := m.store.SaveEnvironment(ctx, msg.Env); err != nil {
			m.err = err
			m.status = fmt.Sprintf("Error saving env: %v", err)
		} else {
			m.status = fmt.Sprintf("Saved environment: %s", msg.Env.Name)
			if m.activeEnv != nil && m.activeEnv.Name == msg.Env.Name {
				env := msg.Env
				m.activeEnv = &env
				m.reqEditor.SetEnvironment(env.Variables)
			}
		}
		return m, nil

	case envpicker.DismissMsg:
		return m, nil

	case tree.RequestSelectedMsg:
		m.reqEditor.SetRequest(msg.Collection, msg.Request)
		return m, nil

	case tree.NewRequestMsg:
		m.newReq.Show(msg.Collection)
		return m, nil

	case newreq.RequestCreatedMsg:
		ctx := context.Background()
		col, err := m.store.LoadCollection(ctx, msg.Collection)
		if err != nil {
			m.err = err
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		col.Requests = append(col.Requests, msg.Request)
		if saveErr := m.store.SaveCollection(ctx, col); saveErr != nil {
			m.err = saveErr
			m.status = fmt.Sprintf("Error: %v", saveErr)
			return m, nil
		}
		m.tree.AddRequest(msg.Collection, msg.Request)
		m.reqEditor.SetRequest(msg.Collection, msg.Request)
		m.status = fmt.Sprintf("Created request: %s", msg.Request.Name)
		return m, nil

	case newreq.CancelledMsg:
		return m, nil

	case reqeditor.SaveRequestMsg:
		ctx := context.Background()
		col, err := m.store.LoadCollection(ctx, msg.Collection)
		if err != nil {
			m.err = err
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		for i, r := range col.Requests {
			if r.Name == msg.Request.Name {
				col.Requests[i] = msg.Request
				break
			}
		}
		if err := m.store.SaveCollection(ctx, col); err != nil {
			m.err = err
			m.status = fmt.Sprintf("Error saving: %v", err)
		} else {
			m.status = fmt.Sprintf("Saved: %s", msg.Request.Name)
		}
		return m, nil

	case reqeditor.SendRequestMsg:
		req := msg.Request
		if m.activeEnv != nil {
			req = envmanager.ResolveRequest(req, *m.activeEnv)
		}
		m.status = "Sending request..."
		return m, m.executeRequest(req)

	case responseMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.respView.SetResponse(msg.resp)
			m.err = nil
			m.status = fmt.Sprintf("Response: %d (%s)", msg.resp.StatusCode, msg.resp.Duration)
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
	contentH := m.height - 1 // reserve 1 row for status bar
	leftW := m.width * 30 / 100
	rightW := m.width - leftW - 1
	topH := contentH * 40 / 100
	bottomH := contentH - topH - 1

	m.tree.SetSize(leftW, contentH)
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

	statusH := 1
	contentH := m.height - statusH

	leftPane := treeStyle.Width(leftW - 2).Height(contentH - 2).Render(m.tree.View())
	rightW := m.width - leftW - 1
	topH := contentH * 40 / 100
	bottomH := contentH - topH - 1

	topPane := reqStyle.Width(rightW - 2).Height(topH - 2).Render(m.reqEditor.View())
	bottomPane := respStyle.Width(rightW - 2).Height(bottomH - 2).Render(m.respView.View())

	rightSide := lipgloss.JoinVertical(lipgloss.Left, topPane, bottomPane)
	contentLayout := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightSide)
	// Clamp content so status bar is always visible
	contentLayout = lipgloss.NewStyle().MaxHeight(contentH).Render(contentLayout)

	// Status bar
	statusFg := lipgloss.Color("252")
	if m.err != nil {
		statusFg = lipgloss.Color("196")
	}
	statusStyle := lipgloss.NewStyle().
		Foreground(statusFg).
		Background(lipgloss.Color("236")).
		Width(m.width)
	statusBar := statusStyle.Render(" " + m.status)
	layout := lipgloss.JoinVertical(lipgloss.Left, contentLayout, statusBar)

	// Overlay help if visible
	if m.helpOverlay.Visible() {
		overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.helpOverlay.View())
		v := tea.NewView(overlay)
		v.AltScreen = true
		return v
	}

	// Overlay env picker if visible
	if m.envPicker.Visible() {
		overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envPicker.View())
		v := tea.NewView(overlay)
		v.AltScreen = true
		return v
	}

	// Overlay new request modal if visible
	if m.newReq.Visible() {
		overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.newReq.View())
		v := tea.NewView(overlay)
		v.AltScreen = true
		return v
	}

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
	p := tea.NewProgram(initialModel(store))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
