# Environment Variable Editor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add KV editing for environment variables inside the existing envpicker modal.

**Architecture:** Extend `envpicker.Model` with a two-screen flow (list → edit) and KV editing state mirroring the reqeditor pattern. Parent handles persistence via `EnvSavedMsg`.

**Tech Stack:** Go, Bubble Tea v2, Lipgloss v2

---

### Task 1: Add types and state to envpicker

**Files:**
- Modify: `internal/tui/envpicker/envpicker.go:1-16`

**Step 1: Add screen, editMode enums, kvRow struct, EnvSavedMsg, and new Model fields**

```go
type EnvSavedMsg struct{ Env domain.Environment }

type screen int
const (
	screenList screen = iota
	screenEdit
)

type editMode int
const (
	editNone editMode = iota
	editKey
	editValue
)

type kvRow struct {
	Key   string
	Value string
}

type Model struct {
	envs     []domain.Environment
	cursor   int
	visible  bool
	screen   screen
	editIdx  int
	kvRows   []kvRow
	kvCursor int
	editMode editMode
	editBuf  string
}
```

**Step 2: Run `make build` — expected: compiles OK**

**Step 3: Commit** `git commit -m "feat(envpicker): add types and state for variable editor"`

---

### Task 2: Implement edit screen Update logic

**Files:**
- Modify: `internal/tui/envpicker/envpicker.go`

**Step 1: Add helper methods**

```go
func (m *Model) enterEditScreen() {
	env := m.envs[m.cursor]
	m.editIdx = m.cursor
	m.screen = screenEdit
	m.kvRows = nil
	for k, v := range env.Variables {
		m.kvRows = append(m.kvRows, kvRow{Key: k, Value: v})
	}
	m.kvCursor = 0
	m.editMode = editNone
}

func (m *Model) buildEnvFromRows() domain.Environment {
	env := m.envs[m.editIdx]
	env.Variables = make(map[string]string)
	for _, row := range m.kvRows {
		if row.Key != "" {
			env.Variables[row.Key] = row.Value
		}
	}
	m.envs[m.editIdx] = env
	return env
}
```

**Step 2: Refactor Update() — route by screen**

In `Update()`, when `m.screen == screenEdit`, handle:
- **editKey/editValue modes:** same pattern as reqeditor's `updateKVKeyMode`/`updateKVValueMode` — Enter commits field (key→value→done+sync), Esc cancels, backspace/text input on editBuf
- **editNone mode:** j/k navigate kvCursor, `a` adds row + enters editKey, `d` deletes row, `e`/Enter edits row (enters editKey with existing key), Esc calls `buildEnvFromRows()` and emits `EnvSavedMsg`, returns to screenList

When `m.screen == screenList`, add `"e"` case that calls `enterEditScreen()`. Existing Enter/Esc/j/k behavior unchanged.

**Step 3: Run `make build` — expected: compiles OK**

**Step 4: Commit** `git commit -m "feat(envpicker): implement variable editor update logic"`

---

### Task 3: Implement edit screen View

**Files:**
- Modify: `internal/tui/envpicker/envpicker.go` (View method)

**Step 1: Refactor View() to branch on screen**

For `screenList`: existing view, add `"e to edit"` to footer hint.

For `screenEdit`: render title `"Edit: <env.Name>"`, then KV rows with cursor prefix `"> "`, showing `editBuf█` when in editKey/editValue mode (same rendering as reqeditor's `renderKVRows`). Footer: `"a add, d delete, e edit, Esc save & back"`.

**Step 2: Run `make run`, test manually: Ctrl+E → e → see variables**

**Step 3: Commit** `git commit -m "feat(envpicker): implement variable editor view"`

---

### Task 4: Wire EnvSavedMsg in main.go

**Files:**
- Modify: `cmd/postmaniux/main.go:153-160`

**Step 1: Add handler after the EnvSelectedMsg case**

```go
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
		}
	}
	return m, nil
```

**Step 2: Run `make build` — expected: compiles OK**

**Step 3: Commit** `git commit -m "feat(main): handle EnvSavedMsg for persistence"`

---

### Task 5: Manual integration test

**Step 1:** Run `make run`
**Step 2:** Ctrl+E → verify env list shows, `e` opens variable editor
**Step 3:** Test add (a), edit (e), delete (d), navigate (j/k)
**Step 4:** Esc from editor → verify status bar shows "Saved environment: X"
**Step 5:** Ctrl+E → e again → verify changes persisted
**Step 6:** Run `make test` and `make lint` — expected: all pass

**Step 7: Commit** `git commit -m "fix: any issues found during integration testing"`

---

### Task 6: Update help overlay

**Files:**
- Modify: `internal/tui/help/help.go` (if it lists keybindings)

**Step 1:** Add `e` keybinding under environment section.

**Step 2: Commit** `git commit -m "docs(help): add env editor keybinding"`
