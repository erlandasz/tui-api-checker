# Design: Create Request Within a Collection

## Summary

Users can create new requests within existing collections from the tree pane. Press `n` to open a modal, type a name, and a placeholder request (GET, empty URL) is added to the selected collection and persisted to disk.

## User Flow

1. User focuses the tree pane and highlights a collection or request node
2. Presses `n`
3. A centered modal overlay appears with a text input for the request name
4. User types a name and presses Enter (or Esc to cancel)
5. A placeholder `Request{Name: name, Method: "GET"}` is created
6. The request is appended to the collection, persisted to storage, added to the tree, and auto-selected in the request editor

## Scope

- Requests only — no collection creation
- Name input only — method/URL/headers edited later in the request editor
- Context-based collection targeting (cursor position determines collection)

## Components

### 1. New Component: `internal/tui/newreq/`

Modal overlay with a single text input.

**Model state:**
- `visible bool`
- `textInput` — Bubble Tea v2 text input
- `collection string` — target collection name

**Public API:**
- `New() Model`
- `Show(collectionName string)` — make visible, set target, focus input
- `Toggle()` / `Visible() bool`
- `SetSize(w, h int)`

**Messages emitted:**
- `RequestCreatedMsg { Collection string, Request domain.Request }` — on Enter with non-empty name
- `CancelledMsg` — on Esc

**Keybindings:**
- Enter — confirm (if name non-empty)
- Esc — cancel
- All other input → text field

**View:** Centered box with title "New Request", text input, hint text (Enter/Esc).

### 2. Tree Changes: `internal/tui/tree/`

**New keybinding:** `n` when focused.

**New message:** `NewRequestMsg { Collection string }` — collection determined from cursor:
- Cursor on collection node → that collection
- Cursor on request node → parent collection

**New method:** `AddRequest(collection string, req domain.Request)` — appends request node, expands collection if collapsed, moves cursor to new node.

### 3. Root Model Changes: `cmd/ratatuile/main.go`

**New state:** `newReq newreq.Model`

**Message routing:**
- `tree.NewRequestMsg` → `newReq.Show(msg.Collection)`
- `newreq.RequestCreatedMsg` → load collection, append request, save, update tree, select in editor
- `newreq.CancelledMsg` → hide modal

**Input capture:** When `newReq.Visible()`, route all input to newreq modal.

**View:** Render newreq overlay on top of layout when visible.

### 4. Help Overlay

Add `n — New request` to the keybinding list.

### 5. Storage & Domain

No changes needed. Existing `LoadCollection` + `SaveCollection` handle persistence.
