# postmaniux

A fast, keyboard-driven TUI API client. Single Go binary, no cloud, no bloat.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)

## Features

- **Collection browser** — tree view to organize requests into folders
- **Request editor** — method, URL, headers, params, JSON body with tabbed sections
- **Response viewer** — color-coded status, pretty-printed JSON, scrollable body
- **Environments** — `{{variable}}` substitution, quick-switch with Ctrl+E
- **Keyboard-first** — no mouse needed, vim-style navigation

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

| Key        | Action             |
|------------|--------------------|
| `Ctrl+W`   | Cycle pane focus   |
| `j` / `k`  | Navigate up/down   |
| `Enter`    | Expand/select      |
| `Tab`      | Switch editor tab  |
| `Ctrl+S`   | Send request       |
| `Ctrl+E`   | Switch environment |
| `?`        | Help overlay       |
| `q`        | Quit               |

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

## Development

```bash
make build   # Build to bin/postmaniux
make run     # Run via go run
make test    # Run all tests
make lint    # go vet
```

## License

MIT
