# Design: Request Editor Editing

## Summary

Make all request fields editable in the request editor pane. Method cycles with `m`, URL edits inline, Headers/Params are structured KV lists with add/edit/delete, Body is a multi-line text area. Vim-style modal editing: view mode by default, explicit enter/exit for each edit mode.

## Modes

- `modeNone` — view/navigation. Existing keybindings work (Tab, Ctrl+S).
- `modeURL` — inline text input on URL line
- `modeKVKey` — editing key of a header/param row
- `modeKVValue` — editing value of a header/param row
- `modeBody` — multi-line text area for body

## Field Focus (modeNone)

Three focusable fields: `fieldMethod`, `fieldURL`, `fieldContent`. Navigate with `j`/`k`. When on fieldContent with Headers/Params tab, `j`/`k` navigates KV rows. First `k` from first KV row moves back to fieldURL.

## New State

- `editMode` — current mode
- `editBuf string` — text buffer for URL/KV editing
- `activeField` — which field is focused
- `kvRows []kvRow` — ordered key-value pairs for active tab
- `kvCursor int` — selected KV row
- `bodyCursorRow, bodyCursorCol int` — cursor in body editor
- `bodyLines []string` — body split into lines

## Method Editing

`m` when on fieldMethod cycles: GET -> POST -> PUT -> PATCH -> DELETE -> GET.

## URL Editing

`e`/Enter on fieldURL enters modeURL. editBuf initialized from current URL. Type/Backspace to edit. Enter confirms, Esc cancels. View shows editBuf + cursor block.

## KV Editing (Headers/Params)

### Data

`kvRows []kvRow` synced from/to request.Headers or request.Params.

### modeNone keybindings (on Headers/Params tab)

- `j`/`k` — navigate rows
- `a` — add new row, enter modeKVKey
- `e`/Enter — edit selected row, enter modeKVKey
- `d` — delete selected row, sync to request

### modeKVKey

Type/Backspace edits key. Enter confirms key, moves to modeKVValue. Esc cancels (removes row if new).

### modeKVValue

Type/Backspace edits value. Enter confirms, syncs to request, returns to modeNone. Esc cancels, returns to modeNone.

## Body Editing

`e`/Enter on fieldContent with Body tab enters modeBody. bodyLines initialized from request.Body. Arrow keys move cursor. Type inserts, Backspace deletes (joins lines at start), Enter splits line. Esc exits, joins bodyLines back to request.Body.

## Files Changed

- `internal/tui/reqeditor/reqeditor.go` — all editing logic
- `internal/tui/help/help.go` — add editing keybindings

No new packages. No changes to domain, storage, or root model.
