# Create Request Within Collection — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let users press `n` in the tree pane to create a new request inside the selected collection via a modal overlay.

**Architecture:** New `internal/tui/newreq/` modal component with hand-rolled text input. Tree emits `NewRequestMsg`, root wires modal + persistence. Follows existing overlay pattern (envpicker/help).

**Tech Stack:** Go 1.25, bubbletea v2 (`charm.land/bubbletea/v2`), lipgloss v2 (`charm.land/lipgloss/v2`)

**Design doc:** `docs/plans/2026-03-06-create-request-design.md`

---

### Task 1: Create `internal/tui/newreq/` modal component

**Files:**
- Create: `internal/tui/newreq/newreq.go`
- Create: `internal/tui/newreq/newreq_test.go`

**Step 1: Write the test file**

```go
package newreq

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/erlandas/ratatuile/internal/domain"
)

func TestNewreq_NotVisibleByDefault(t *testing.T) {
	m := New()
	if m.Visible() {
		t.Error("expected not visible by default")
	}
}

func TestNewreq_ShowMakesVisible(t *testing.T) {
	m := New()
	m.Show("my-collection")
	if !m.Visible() {
		t.Error("expected visible after Show")
	}
}

func TestNewreq_EscCancels(t *testing.T) {
	m := New()
	m.Show("col")
	updated, cmd := m.Update(tea.KeyPressMsg{Key: tea.Key{Code: tea.KeyEscape}})
	if updated.Visible() {
		t.Error("expected not visible after Esc")
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	if _, ok := msg.(CancelledMsg); !ok {
		t.Errorf("expected CancelledMsg, got %T", msg)
	}
}

func TestNewreq_EnterWithEmptyNameDoesNothing(t *testing.T) {
	m := New()
	m.Show("col")
	updated, cmd := m.Update(tea.KeyPressMsg{Key: tea.Key{Code: tea.KeyEnter}})
	if !updated.Visible() {
		t.Error("expected still visible with empty name")
	}
	if cmd != nil {
		t.Error("expected no command with empty name")
	}
}

func TestNewreq_TypeAndEnterCreatesRequest(t *testing.T) {
	m := New()
	m.Show("my-api")

	// Type "Get Users" character by character
	for _, ch := range "Get Users" {
		m, _ = m.Update(tea.KeyPressMsg{Key: tea.Key{Code: ch, Text: string(ch)}})
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Key: tea.Key{Code: tea.KeyEnter}})
	if updated.Visible() {
		t.Error("expected not visible after confirm")
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	created, ok := msg.(RequestCreatedMsg)
	if !ok {
		t.Fatalf("expected RequestCreatedMsg, got %T", msg)
	}
	if created.Collection != "my-api" {
		t.Errorf("collection = %q, want %q", created.Collection, "my-api")
	}
	if created.Request.Name != "Get Users" {
		t.Errorf("name = %q, want %q", created.Request.Name, "Get Users")
	}
	if created.Request.Method != "GET" {
		t.Errorf("method = %q, want %q", created.Request.Method, "GET")
	}
}

func TestNewreq_BackspaceDeletesChar(t *testing.T) {
	m := New()
	m.Show("col")

	for _, ch := range "abc" {
		m, _ = m.Update(tea.KeyPressMsg{Key: tea.Key{Code: ch, Text: string(ch)}})
	}
	m, _ = m.Update(tea.KeyPressMsg{Key: tea.Key{Code: tea.KeyBackspace}})
	m, cmd := m.Update(tea.KeyPressMsg{Key: tea.Key{Code: tea.KeyEnter}})

	msg := cmd()
	created := msg.(RequestCreatedMsg)
	if created.Request.Name != "ab" {
		t.Errorf("name = %q, want %q", created.Request.Name, "ab")
	}
	_ = m
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/newreq/ -v`
Expected: compilation error (package doesn't exist yet)

**Step 3: Write the implementation**

```go
package newreq

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/domain"
)

type RequestCreatedMsg struct {
	Collection string
	Request    domain.Request
}

type CancelledMsg struct{}

type Model struct {
	visible    bool
	input      string
	collection string
}

func New() Model { return Model{} }

func (m *Model) Show(collection string) {
	m.visible = true
	m.collection = collection
	m.input = ""
}

func (m *Model) Toggle()      { m.visible = !m.visible }
func (m Model) Visible() bool { return m.visible }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "escape":
			m.visible = false
			return m, func() tea.Msg { return CancelledMsg{} }
		case "enter":
			if m.input == "" {
				return m, nil
			}
			req := domain.Request{Name: m.input, Method: "GET"}
			col := m.collection
			m.visible = false
			m.input = ""
			return m, func() tea.Msg {
				return RequestCreatedMsg{Collection: col, Request: req}
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			key := msg.Key()
			if key.Text != "" {
				m.input += key.Text
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	s := title.Render("New Request") + "\n\n"
	s += "Name: " + m.input + "█\n\n"
	s += dim.Render("Enter to confirm, Esc to cancel")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/newreq/ -v`
Expected: all 6 tests PASS

**Step 5: Commit**

```bash
git add internal/tui/newreq/
git commit -m "feat: add newreq modal component for creating requests"
```

---

### Task 2: Add `NewRequestMsg` and `n` keybinding to tree component

**Files:**
- Modify: `internal/tui/tree/tree.go:31-34` (add message type) and `:73-98` (add keybinding)

**Step 1: Add `NewRequestMsg` type after `RequestSelectedMsg` (line 34)**

```go
type NewRequestMsg struct {
	Collection string
}
```

**Step 2: Add `n` keybinding in the `Update` key switch (after `"enter"` case, before closing `}`)**

Inside `case tea.KeyPressMsg:`, after the `"enter"` case block (line 97), add:

```go
case "n":
	visible := m.visibleNodes()
	if m.cursor < len(visible) {
		node := visible[m.cursor]
		col := node.Collection
		return m, func() tea.Msg {
			return NewRequestMsg{Collection: col}
		}
	}
```

**Step 3: Add `AddRequest` method**

Append to the file:

```go
func (m *Model) AddRequest(collection string, req domain.Request) {
	// Find insertion point: after last node of this collection
	insertIdx := -1
	folderIdx := -1
	for i, n := range m.nodes {
		if n.IsFolder && n.Collection == collection {
			folderIdx = i
			m.nodes[i].Expanded = true
			insertIdx = i + 1
		} else if folderIdx >= 0 && n.Collection == collection {
			insertIdx = i + 1
		} else if folderIdx >= 0 && n.Collection != collection {
			break
		}
	}
	if insertIdx < 0 {
		return
	}

	newNode := Node{
		Name:       req.Name,
		IsFolder:   false,
		Depth:      1,
		Collection: collection,
		Request:    &req,
	}

	// Insert node at position
	m.nodes = append(m.nodes, Node{})
	copy(m.nodes[insertIdx+1:], m.nodes[insertIdx:])
	m.nodes[insertIdx] = newNode

	// Move cursor to new node in visible list
	for i, n := range m.visibleNodes() {
		if !n.IsFolder && n.Request != nil && n.Request.Name == req.Name && n.Collection == collection {
			m.cursor = i
			break
		}
	}
}
```

**Step 4: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/tree/tree.go
git commit -m "feat: add n keybinding and AddRequest to tree component"
```

---

### Task 3: Wire newreq modal into root model

**Files:**
- Modify: `cmd/ratatuile/main.go`

**Step 1: Add import**

Add to imports (line 16):
```go
"github.com/erlandas/ratatuile/internal/tui/newreq"
```

**Step 2: Add field to `model` struct (after line 37)**

```go
newReq newreq.Model
```

**Step 3: Initialize in `initialModel` (after line 79)**

```go
newReq: newreq.New(),
```

**Step 4: Add input capture for newreq modal in `Update`**

After the envPicker visible check (line 109), add:

```go
// When new request modal is visible, delegate all input to it
if m.newReq.Visible() {
	var cmd tea.Cmd
	m.newReq, cmd = m.newReq.Update(msg)
	return m, cmd
}
```

**Step 5: Add message handlers**

After the `tree.RequestSelectedMsg` case (line 139), add:

```go
case tree.NewRequestMsg:
	m.newReq.Show(msg.Collection)
	return m, nil

case newreq.RequestCreatedMsg:
	ctx := context.Background()
	col, err := m.store.LoadCollection(ctx, msg.Collection)
	if err == nil {
		col.Requests = append(col.Requests, msg.Request)
		if saveErr := m.store.SaveCollection(ctx, col); saveErr != nil {
			m.err = saveErr
		}
	} else {
		m.err = err
	}
	m.tree.AddRequest(msg.Collection, msg.Request)
	m.reqEditor.SetRequest(msg.Request)
	return m, nil

case newreq.CancelledMsg:
	return m, nil
```

**Step 6: Add overlay rendering in `View`**

After the envPicker overlay block (line 268), add:

```go
// Overlay new request modal if visible
if m.newReq.Visible() {
	overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.newReq.View())
	v := tea.NewView(overlay)
	v.AltScreen = true
	return v
}
```

**Step 7: Build and verify**

Run: `go build ./cmd/ratatuile/`
Expected: compiles without errors

**Step 8: Run all tests**

Run: `go test ./... -v`
Expected: all PASS

**Step 9: Commit**

```bash
git add cmd/ratatuile/main.go
git commit -m "feat: wire newreq modal into root model with persistence"
```

---

### Task 4: Add keybinding to help overlay

**Files:**
- Modify: `internal/tui/help/help.go:46-54`

**Step 1: Add new entry to bindings slice**

In the `bindings` slice (line 46), add before the `{"q / Ctrl+C", "Quit"}` entry:

```go
{"n", "New request (in tree)"},
```

**Step 2: Run all tests**

Run: `go test ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/tui/help/help.go
git commit -m "feat: add n keybinding to help overlay"
```

---

### Task 5: Manual smoke test

**Steps:**
1. Create a test collection: `mkdir -p ~/.ratatuile/collections/demo && echo '{"name":"demo","requests":[]}' > ~/.ratatuile/collections/demo/collection.json`
2. Run: `go run ./cmd/ratatuile/`
3. Verify tree shows "demo" collection
4. Press `n` — modal should appear
5. Type "Hello World" and press Enter
6. Verify request appears in tree under demo, and request editor shows "GET Hello World"
7. Press `?` — verify help shows "n  New request (in tree)"
8. Quit and re-run — verify the request persists
