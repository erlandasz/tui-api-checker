# Request Editor Editing — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make all request fields editable in the request editor pane.

**Architecture:** Add modal editing to the existing reqeditor component. New types (editMode, field, kvRow), new state fields, updated Update/View. No new packages.

**Tech Stack:** Go 1.25, bubbletea v2, lipgloss v2

**Design doc:** `docs/plans/2026-03-06-reqeditor-editing-design.md`

---

### Task 1: Add types, state fields, and field navigation

**Files:**
- Modify: `internal/tui/reqeditor/reqeditor.go`

**What to build:**

Add these types before the Model struct:

```go
type editMode int

const (
	modeNone editMode = iota
	modeURL
	modeKVKey
	modeKVValue
	modeBody
)

type field int

const (
	fieldMethod field = iota
	fieldURL
	fieldContent
)

type kvRow struct {
	Key   string
	Value string
}
```

Add these fields to Model:

```go
type Model struct {
	request        domain.Request
	activeTab      Tab
	focused        bool
	width          int
	height         int
	editMode       editMode
	activeField    field
	editBuf        string
	kvRows         []kvRow
	kvCursor       int
	bodyLines      []string
	bodyCursorRow  int
	bodyCursorCol  int
}
```

Add field navigation in Update for modeNone (inside the existing KeyPressMsg switch, before the `"tab"` case):

```go
case "j", "down":
	if m.activeField < fieldContent {
		m.activeField++
	}
case "k", "up":
	if m.activeField > fieldMethod {
		m.activeField--
	}
```

Update View to show a cursor indicator `>` next to the active field.

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(reqeditor): add edit types, state fields, and field navigation"`

---

### Task 2: Method cycling

**Files:**
- Modify: `internal/tui/reqeditor/reqeditor.go`

**What to build:**

Add a methods slice at package level:

```go
var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
```

In Update modeNone, add:

```go
case "m":
	if m.activeField == fieldMethod {
		current := m.request.Method
		for i, method := range httpMethods {
			if method == current {
				m.request.Method = httpMethods[(i+1)%len(httpMethods)]
				break
			}
		}
		if m.request.Method == current {
			m.request.Method = httpMethods[0]
		}
	}
```

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(reqeditor): add method cycling with m key"`

---

### Task 3: URL inline editing

**Files:**
- Modify: `internal/tui/reqeditor/reqeditor.go`

**What to build:**

In Update modeNone, add:

```go
case "e", "enter":
	switch m.activeField {
	case fieldURL:
		m.editMode = modeURL
		m.editBuf = m.request.URL
	}
```

Add a new section in Update for modeURL:

```go
if m.editMode == modeURL {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.request.URL = m.editBuf
			m.editMode = modeNone
		case "esc", "escape":
			m.editMode = modeNone
		case "backspace":
			if len(m.editBuf) > 0 {
				m.editBuf = m.editBuf[:len(m.editBuf)-1]
			}
		default:
			if key := msg.Key(); key.Text != "" {
				m.editBuf += key.Text
			}
		}
	}
	return m, nil
}
```

Update View: when `m.editMode == modeURL`, render URL line as `METHOD editBuf█` instead of `METHOD request.URL`.

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(reqeditor): add inline URL editing"`

---

### Task 4: KV list display and navigation

**Files:**
- Modify: `internal/tui/reqeditor/reqeditor.go`

**What to build:**

Add helper methods to sync kvRows from/to request maps:

```go
func (m *Model) syncKVFromRequest() {
	m.kvRows = nil
	var src map[string]string
	switch m.activeTab {
	case TabHeaders:
		src = m.request.Headers
	case TabParams:
		src = m.request.Params
	}
	for k, v := range src {
		m.kvRows = append(m.kvRows, kvRow{Key: k, Value: v})
	}
	m.kvCursor = 0
}

func (m *Model) syncKVToRequest() {
	kv := make(map[string]string)
	for _, row := range m.kvRows {
		if row.Key != "" {
			kv[row.Key] = row.Value
		}
	}
	switch m.activeTab {
	case TabHeaders:
		m.request.Headers = kv
	case TabParams:
		m.request.Params = kv
	}
}
```

Call `syncKVFromRequest()` when switching tabs (in the `"tab"` and `"shift+tab"` cases) and in `SetRequest`.

When `activeField == fieldContent` on Headers/Params tab, `j`/`k` navigates `kvCursor` within kvRows instead of moving between fields.

Update `renderMap` to show cursor on the active row when focused.

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(reqeditor): add KV list display and row navigation"`

---

### Task 5: KV add, edit, delete

**Files:**
- Modify: `internal/tui/reqeditor/reqeditor.go`

**What to build:**

In modeNone when `activeField == fieldContent` and tab is Headers or Params:

```go
case "a":
	m.kvRows = append(m.kvRows, kvRow{})
	m.kvCursor = len(m.kvRows) - 1
	m.editMode = modeKVKey
	m.editBuf = ""
case "e", "enter":
	if len(m.kvRows) > 0 {
		m.editMode = modeKVKey
		m.editBuf = m.kvRows[m.kvCursor].Key
	}
case "d":
	if len(m.kvRows) > 0 {
		m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
		if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
			m.kvCursor--
		}
		m.syncKVToRequest()
	}
```

Add modeKVKey handler:

```go
case "enter":
	m.kvRows[m.kvCursor].Key = m.editBuf
	m.editMode = modeKVValue
	m.editBuf = m.kvRows[m.kvCursor].Value
case "esc", "escape":
	// If new empty row, remove it
	if m.kvRows[m.kvCursor].Key == "" && m.kvRows[m.kvCursor].Value == "" {
		m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
		if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
			m.kvCursor--
		}
	}
	m.editMode = modeNone
```

Add modeKVValue handler:

```go
case "enter":
	m.kvRows[m.kvCursor].Value = m.editBuf
	m.syncKVToRequest()
	m.editMode = modeNone
case "esc", "escape":
	m.editMode = modeNone
```

Both KV modes handle backspace/typing same as URL mode.

Update View to show edit buffer with cursor when editing a KV row.

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(reqeditor): add KV add, edit, delete"`

---

### Task 6: Body text area editing

**Files:**
- Modify: `internal/tui/reqeditor/reqeditor.go`

**What to build:**

Add body sync helpers:

```go
func (m *Model) syncBodyFromRequest() {
	if m.request.Body == "" {
		m.bodyLines = []string{""}
	} else {
		m.bodyLines = strings.Split(m.request.Body, "\n")
	}
	m.bodyCursorRow = 0
	m.bodyCursorCol = 0
}

func (m *Model) syncBodyToRequest() {
	m.request.Body = strings.Join(m.bodyLines, "\n")
}
```

Call `syncBodyFromRequest()` when switching to Body tab and in `SetRequest`.

Enter modeBody via `e`/Enter on fieldContent when tab is Body.

Add modeBody handler:

```go
case "esc", "escape":
	m.syncBodyToRequest()
	m.editMode = modeNone
case "enter":
	// Split line at cursor
	line := m.bodyLines[m.bodyCursorRow]
	m.bodyLines[m.bodyCursorRow] = line[:m.bodyCursorCol]
	rest := line[m.bodyCursorCol:]
	m.bodyLines = append(m.bodyLines[:m.bodyCursorRow+1], append([]string{rest}, m.bodyLines[m.bodyCursorRow+1:]...)...)
	m.bodyCursorRow++
	m.bodyCursorCol = 0
case "backspace":
	if m.bodyCursorCol > 0 {
		line := m.bodyLines[m.bodyCursorRow]
		m.bodyLines[m.bodyCursorRow] = line[:m.bodyCursorCol-1] + line[m.bodyCursorCol:]
		m.bodyCursorCol--
	} else if m.bodyCursorRow > 0 {
		// Join with previous line
		prev := m.bodyLines[m.bodyCursorRow-1]
		m.bodyCursorCol = len(prev)
		m.bodyLines[m.bodyCursorRow-1] = prev + m.bodyLines[m.bodyCursorRow]
		m.bodyLines = append(m.bodyLines[:m.bodyCursorRow], m.bodyLines[m.bodyCursorRow+1:]...)
		m.bodyCursorRow--
	}
case "left":
	if m.bodyCursorCol > 0 {
		m.bodyCursorCol--
	}
case "right":
	if m.bodyCursorCol < len(m.bodyLines[m.bodyCursorRow]) {
		m.bodyCursorCol++
	}
case "up":
	if m.bodyCursorRow > 0 {
		m.bodyCursorRow--
		if m.bodyCursorCol > len(m.bodyLines[m.bodyCursorRow]) {
			m.bodyCursorCol = len(m.bodyLines[m.bodyCursorRow])
		}
	}
case "down":
	if m.bodyCursorRow < len(m.bodyLines)-1 {
		m.bodyCursorRow++
		if m.bodyCursorCol > len(m.bodyLines[m.bodyCursorRow]) {
			m.bodyCursorCol = len(m.bodyLines[m.bodyCursorRow])
		}
	}
default:
	if key := msg.Key(); key.Text != "" {
		line := m.bodyLines[m.bodyCursorRow]
		m.bodyLines[m.bodyCursorRow] = line[:m.bodyCursorCol] + key.Text + line[m.bodyCursorCol:]
		m.bodyCursorCol += len(key.Text)
	}
```

Update View to render bodyLines with cursor when in modeBody.

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(reqeditor): add body text area editing"`

---

### Task 7: Update help overlay

**Files:**
- Modify: `internal/tui/help/help.go`

Add these entries to the bindings slice:

```go
{"m", "Cycle HTTP method"},
{"e / Enter", "Edit focused field"},
{"a", "Add header/param"},
{"d", "Delete header/param"},
{"Esc", "Cancel / exit edit mode"},
```

**Run:** `go build ./cmd/postmaniux/ && go test ./...`

**Commit:** `git commit -m "feat(help): add request editing keybindings"`

---

### Task 8: Smoke test

1. Run `go run ./cmd/postmaniux/`
2. Select a request in tree, Ctrl+W to focus request editor
3. Verify j/k moves between Method/URL/Content
4. Press `m` on Method — cycles through GET/POST/PUT/PATCH/DELETE
5. Press `e` on URL — type a URL, Enter to confirm
6. Tab to Headers, press `a` — type key, Enter, type value, Enter
7. Verify header appears, press `d` to delete it
8. Tab to Body, press `e` — type text, Enter for newlines, Esc to exit
9. Ctrl+S sends with edited values
