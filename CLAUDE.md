# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

postmaniux is a keyboard-driven TUI API client (Postman alternative) built as a single Go binary. No cloud, no Electron. Uses Bubble Tea v2 for the terminal UI.

## Commands

```bash
make build      # Build binary to bin/postmaniux
make run        # Run directly via go run
make test       # Run all tests: go test ./...
make lint       # Lint: go vet ./...

# Run a single test
go test ./internal/storage/ -run TestSaveAndLoad
```

## Architecture

Three-layer design: **TUI → Core → Storage**

- `cmd/postmaniux/main.go` — Entry point and root Bubble Tea model. Owns the three-pane layout, focus cycling (Ctrl+W), and message routing between child components.
- `internal/domain/` — Core types: `Request`, `Collection`, `Environment`. No dependencies on TUI or storage.
- `internal/storage/` — `FileStore` persists collections/environments as JSON under `~/.postmaniux/`. Collections stored as `collections/{name}/collection.json`, environments as `environments/{name}.json`.
- `internal/httpclient/` — Wraps `net/http` to execute `domain.Request` and return a `Response` with status, headers, body, duration, size.
- `internal/envmanager/` — `{{variable}}` template resolution. `ResolveRequest()` substitutes placeholders from active environment before sending.
- `internal/tui/` — Each sub-package is a self-contained Bubble Tea component:
  - `tree/` — Collection browser (left pane, 30% width). Flat node list with folder expand/collapse.
  - `reqeditor/` — Request editor (top-right, 40% height). Tabbed: Headers, Params, Body. Ctrl+S sends.
  - `respview/` — Response viewer (bottom-right). Color-coded status, pretty-printed JSON, scrollable.
  - `envpicker/` — Modal overlay for switching environments (Ctrl+E).
  - `help/` — Modal overlay showing keybindings (?).

## TUI Component Pattern

All TUI components follow the same Bubble Tea convention:
- Exported `Model` struct with `Init()`, `Update()`, `View()` methods
- `SetFocused(bool)` / `Focused() bool` for focus management
- `SetSize(w, h int)` for layout
- Components communicate to parent via message types (e.g., `tree.RequestSelectedMsg`, `reqeditor.SendRequestMsg`, `envpicker.EnvSelectedMsg`)
- Modal overlays (help, envpicker) use `Toggle()` / `Visible()` and capture all input when visible

## Key Dependencies

- `charm.land/bubbletea/v2` — TUI framework (v2, not v1)
- `charm.land/lipgloss/v2` — Terminal styling (v2)
- Go 1.25.0
