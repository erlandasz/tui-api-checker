package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

// Why: model is the root application state for bubbletea's MVU architecture.
// All child components will be fields on this struct.
type model struct{}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView("postmaniux - press q to quit\n")
}

func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
