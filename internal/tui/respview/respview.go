package respview

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/httpclient"
)

type Model struct {
	response *httpclient.Response
	focused  bool
	scroll   int
	width    int
	height   int
}

func New() Model { return Model{} }

func (m *Model) SetResponse(r *httpclient.Response) {
	m.response = r
	m.scroll = 0
}

func (m *Model) SetSize(w, h int)  { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool) { m.focused = f }
func (m Model) Focused() bool      { return m.focused }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	s := titleStyle.Render("Response") + "\n"

	if m.response == nil {
		s += dimStyle.Render("Send a request with Ctrl+S")
		return s
	}

	r := m.response

	// Status line with color coding
	statusStyle := lipgloss.NewStyle().Bold(true)
	switch {
	case r.StatusCode >= 200 && r.StatusCode < 300:
		statusStyle = statusStyle.Foreground(lipgloss.Color("86"))
	case r.StatusCode >= 300 && r.StatusCode < 400:
		statusStyle = statusStyle.Foreground(lipgloss.Color("228"))
	default:
		statusStyle = statusStyle.Foreground(lipgloss.Color("196"))
	}

	s += statusStyle.Render(fmt.Sprintf("%d", r.StatusCode))
	s += dimStyle.Render(fmt.Sprintf("  %s  %d bytes", r.Duration.Round(1e6), r.Size))
	s += "\n\n"

	// Headers
	s += dimStyle.Render("Headers:") + "\n"
	for k, v := range r.Headers {
		s += fmt.Sprintf("  %s: %s\n", k, v)
	}
	s += dimStyle.Render(strings.Repeat("─", 30)) + "\n"

	// Body — try to pretty-print JSON
	body := r.Body
	var js json.RawMessage
	if json.Unmarshal([]byte(body), &js) == nil {
		if pretty, err := json.MarshalIndent(js, "", "  "); err == nil {
			body = string(pretty)
		}
	}

	lines := strings.Split(body, "\n")
	// Simple scroll by skipping lines. Clamp to avoid overscroll.
	if m.scroll > len(lines) {
		m.scroll = len(lines)
	}
	visible := lines[m.scroll:]
	if m.height > 0 && len(visible) > m.height-8 {
		visible = visible[:m.height-8]
	}

	for i, line := range visible {
		visible[i] = colorizeJSON(line)
	}
	s += strings.Join(visible, "\n")

	return s
}

var (
	jsonKeyRe    = regexp.MustCompile(`^(\s*)("(?:[^"\\]|\\.)*")(\s*:)`)
	jsonStringRe = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	jsonNumberRe = regexp.MustCompile(`\b-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?\b`)
	jsonBoolRe   = regexp.MustCompile(`\b(?:true|false)\b`)
	jsonNullRe   = regexp.MustCompile(`\bnull\b`)

	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))  // cyan — keys
	strStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange — string values
	numStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114")) // green — numbers
	boolStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("177")) // purple — booleans
	nullStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // red — null
	bracketStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // light gray — brackets
)

func colorizeJSON(line string) string {
	// Color the key portion first: "key":
	line = jsonKeyRe.ReplaceAllStringFunc(line, func(match string) string {
		parts := jsonKeyRe.FindStringSubmatch(match)
		if len(parts) == 4 {
			return parts[1] + keyStyle.Render(parts[2]) + parts[3]
		}
		return match
	})

	// Color values after the colon (or standalone values in arrays)
	// Process from right to left to avoid offset issues — but regex handles it fine
	line = jsonNullRe.ReplaceAllStringFunc(line, func(m string) string {
		return nullStyle.Render(m)
	})
	line = jsonBoolRe.ReplaceAllStringFunc(line, func(m string) string {
		return boolStyle.Render(m)
	})
	line = jsonNumberRe.ReplaceAllStringFunc(line, func(m string) string {
		// Don't colorize numbers inside ANSI escape sequences
		if strings.Contains(m, "\x1b") {
			return m
		}
		return numStyle.Render(m)
	})

	// Color remaining uncolored strings (values, not keys which are already colored)
	line = jsonStringRe.ReplaceAllStringFunc(line, func(m string) string {
		// Skip already-colored strings (contain ANSI escape)
		if strings.Contains(m, "\x1b") {
			return m
		}
		return strStyle.Render(m)
	})

	return line
}
