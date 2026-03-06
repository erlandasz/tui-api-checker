# ratatuile - TUI API Client

A fast, keyboard-driven alternative to Postman. Single Go binary, no cloud, no bloat.

## Architecture

Three layers:

```
TUI Layer (bubbletea + lipgloss + huh)
Core Layer (net/http, env/variable manager)
Storage Layer (JSON file I/O)
```

## Layout

Three-pane multi-panel layout:

```
+---------------+-------------------------------+
|               |  Request                      |
|  Collections  |  Method | URL                 |
|  (tree view)  |  Headers / Params / Body      |
|               +-------------------------------+
|               |  Response                     |
|               |  Status | Time | Size         |
|               |  Headers / Body (formatted)   |
+---------------+-------------------------------+
```

- Left pane (30%): collection tree
- Top-right (40% height): request editor with tabbed sections
- Bottom-right (60% height): response viewer with syntax-highlighted JSON

## Data Format

JSON files stored in `~/.ratatuile/`:

```
~/.ratatuile/
  collections/
    my-api/
      collection.json
      get-users.json
      create-user.json
  environments/
    dev.json
    prod.json
  config.json
```

### Request file

```json
{
  "name": "Get Users",
  "method": "GET",
  "url": "{{base_url}}/users",
  "headers": {
    "Authorization": "Bearer {{token}}",
    "Content-Type": "application/json"
  },
  "params": {
    "page": "1",
    "limit": "20"
  }
}
```

### Environment file

```json
{
  "name": "dev",
  "variables": {
    "base_url": "http://localhost:3000",
    "token": "dev-token-123"
  }
}
```

`{{variable}}` syntax resolved at request time from active environment.

## Key Bindings

| Key      | Action              |
|----------|---------------------|
| Ctrl+W   | Cycle pane focus    |
| j/k      | Navigate up/down    |
| Enter    | Expand/select       |
| Ctrl+S   | Send request        |
| Ctrl+E   | Switch environment  |
| Ctrl+N   | New request         |
| Ctrl+D   | Delete request      |
| ?        | Help overlay        |
| q/Ctrl+C | Quit                |

## MVP Features

1. Collection browser - tree view, create/edit/delete requests and folders
2. Request editor - method, URL, headers, params, JSON body (huh forms)
3. Request execution - send via net/http, show response
4. Response viewer - syntax-highlighted JSON, raw body, response headers
5. Environment switching - quick-switch via Ctrl+E, variable substitution
6. Help overlay - keybinding reference toggled with ?

## Tech Stack

- Go
- bubbletea (TUI framework)
- lipgloss (styling)
- huh (form inputs)
- bubbles (common components)
- net/http (HTTP client)

## Not in MVP

- Request history/log
- Import from Postman/curl
- Auth helpers (OAuth, etc.)
- Pre/post request scripts
- Response body search/filter
