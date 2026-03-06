# postmaniux

A fast, keyboard-driven TUI API client. Single Go binary, no cloud, no bloat.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)

## Features

- **Collection browser** — tree view with nested sub-folders, expand/collapse with Enter
- **Request editor** — method, URL, headers, params, JSON body with tabbed sections
- **Response viewer** — color-coded status, pretty-printed JSON, scrollable body
- **Environments** — `{{variable}}` substitution, quick-switch with Ctrl+E, inline variable editor
- **Built-in date variables** — `{{$today}}`, `{{$startOfWeek}}`, etc. auto-resolved
- **Variable autocomplete** — type `{{` to get a dropdown of matching variables, Tab to insert
- **Variable highlighting** — known variables shown in green, unknown in red
- **Persistence** — Ctrl+S saves request edits to disk
- **Keyboard-first** — no mouse needed, vim-style navigation, arrow keys in edit fields

## Layout

```
┌───────────────┬───────────────────────────────┐
│               │  Request                      │
│  Collections  │  GET https://api.example.com  │
│  (tree view)  │  Headers │ Params │ Body      │
│               ├───────────────────────────────┤
│               │  Response                     │
│               │  200  12ms  1.4KB             │
│               │  { "users": [...] }           │
└───────────────┴───────────────────────────────┘
```

## Install

```bash
go install github.com/erlandas/postmaniux/cmd/postmaniux@latest
```

Or build from source:

```bash
git clone https://github.com/erlandas/postmaniux.git
cd postmaniux
make build
./bin/postmaniux
```

## Keybindings

### Global

| Key          | Action                              |
|--------------|-------------------------------------|
| `Ctrl+W`     | Cycle pane focus                    |
| `Ctrl+E`     | Switch environment (e to edit vars) |
| `?`          | Help overlay                        |
| `q` / `Ctrl+C` | Quit                             |

### Navigation

| Key          | Action             |
|--------------|--------------------|
| `j` / `k`   | Navigate up/down   |
| `Enter`      | Expand/select      |
| `Tab`        | Switch editor tab  |
| `n`          | New request (tree) |

### Editing

| Key          | Action                          |
|--------------|---------------------------------|
| `e` / `Enter`| Edit focused field              |
| `a`          | Add header/param                |
| `d`          | Delete header/param             |
| `m`          | Cycle HTTP method               |
| `Left`/`Right`| Move cursor in edit field      |
| `Esc`        | Cancel / exit edit mode         |
| `Ctrl+S`     | Save request to disk            |
| `Ctrl+Enter` | Send request                    |

### Autocomplete (in edit fields)

| Key        | Action                              |
|------------|-------------------------------------|
| `{{`       | Opens variable dropdown             |
| `Up`/`Down`| Navigate dropdown                   |
| `Tab`      | Insert selected variable            |

## Data Storage

All data is stored as JSON files in `~/.postmaniux/`:

```
~/.postmaniux/
  collections/
    my-api/
      collection.json
  environments/
    dev.json
    prod.json
```

### Environment variables

Use `{{variable}}` syntax in URLs, headers, params, and body. Variables are resolved from the active environment at request time.

```json
{
  "name": "dev",
  "variables": {
    "base_url": "http://localhost:3000",
    "token": "dev-token-123"
  }
}
```

### Built-in date variables

These resolve automatically — no environment setup needed:

| Variable | Example | Description |
|---|---|---|
| `{{$today}}` | `2026-03-06` | Current date |
| `{{$yesterday}}` | `2026-03-05` | Yesterday |
| `{{$tomorrow}}` | `2026-03-07` | Tomorrow |
| `{{$startOfWeek}}` | `2026-03-02` | Monday of current week |
| `{{$endOfWeek}}` | `2026-03-08` | Sunday of current week |
| `{{$startOfMonth}}` | `2026-03-01` | First day of month |
| `{{$endOfMonth}}` | `2026-03-31` | Last day of month |

Use them anywhere you use regular variables: `https://api.example.com/reports?from={{$startOfWeek}}&to={{$endOfWeek}}`

## Importing from Postman

Export your Postman data (Settings → Data → Export Data), then run:

```bash
go run scripts/import_postman.go /path/to/postman-export-dir
```

The export directory should contain `collection/` and/or `environment/` subdirectories with JSON files. The script imports both collections (with nested folder structure preserved) and environments into `~/.postmaniux/`.

## Development

```bash
make build   # Build to bin/postmaniux
make run     # Run via go run
make test    # Run all tests
make lint    # go vet
```

## License

MIT
