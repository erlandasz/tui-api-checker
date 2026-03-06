# Environment Variable Editor

## Goal

Allow users to edit environment variable key-value pairs (add, edit, delete) from within the existing envpicker modal. No new keybindings or panes needed.

## UX Flow

1. Ctrl+E opens the env picker (existing)
2. `e` on a highlighted environment opens the variable editor for that environment
3. Enter on an environment still activates it (existing behavior)
4. Variable editor shows KV rows using the same interaction pattern as reqeditor:
   - j/k — navigate rows
   - a — add new variable (key edit → value edit)
   - d — delete selected variable
   - e/Enter — edit selected variable (key → value)
   - Esc — auto-save to disk, return to environment list
5. Esc from environment list closes the modal (existing)

## Architecture

All changes scoped to two files:

### `internal/tui/envpicker/envpicker.go`

- Add `screen` enum: `screenList` / `screenEdit`
- Add KV editing state: `kvRows []kvRow`, `kvCursor int`, `editMode`, `editBuf string`
- `editMode` enum: `editNone`, `editKey`, `editValue`
- Track which environment is being edited: `editIdx int`
- `Update()` routes input based on active screen
- `View()` renders either the env list or the variable editor
- On Esc from `screenEdit`: emit `EnvSavedMsg` with updated environment

### `cmd/postmaniux/main.go`

- Handle `EnvSavedMsg`: call `store.SaveEnvironment()`, update `activeEnv` if it matches

### No changes to

- `domain/` — Environment struct already has `map[string]string` for variables
- `storage/` — `SaveEnvironment()` already exists
- `envmanager/` — resolution logic unchanged

## Message Types

- `EnvSavedMsg{ Env domain.Environment }` — emitted when user exits variable editor

## Keybinding Summary (envpicker modal)

| Screen | Key | Action |
|--------|-----|--------|
| List | j/k | Navigate environments |
| List | Enter | Activate environment |
| List | e | Edit environment variables |
| List | Esc | Close modal |
| Edit | j/k | Navigate variable rows |
| Edit | a | Add new variable |
| Edit | d | Delete selected variable |
| Edit | e/Enter | Edit selected variable |
| Edit | Esc | Save and return to list |
| Edit (editing) | Enter | Confirm field (key→value, value→done) |
| Edit (editing) | Esc | Cancel edit, return to navigation |
