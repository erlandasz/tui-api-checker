# Sub-Collection Tree Display

## Goal

Parse ` / ` separators in request names to display a nested folder hierarchy in the tree component. Display-only change — no domain model modifications.

## How It Works

Request named `Auth / Login / Get Token` renders as nested folders:
```
v Collection
    v Auth
        v Login
            GET Get Token
```

Leaf nodes display only the final name segment. Sub-folders are expandable/collapsible with Enter, same as top-level collections.

## Architecture

**Only file changed:** `internal/tui/tree/tree.go`

### Node identity

Add a `Path` field to `Node` — the full folder path (e.g., `"Auth/Login"`). Top-level collection folders use `""`. This is used for toggle matching and visibility logic.

### New() changes

For each request, split name by ` / `. For each prefix segment, create a folder node if one doesn't already exist at that path. The request node uses the last segment as its display name.

Example: `"Auth / Login / Get Token"` produces:
- Folder node: Name=`Auth`, Depth=1, Path=`Auth`, Collection=`col`
- Folder node: Name=`Login`, Depth=2, Path=`Auth/Login`, Collection=`col`
- Request node: Name=`Get Token`, Depth=3, Path=`Auth/Login`, Collection=`col`

### visibleNodes() changes

Replace the simple `showChildren` bool with depth-aware logic: track a `hideBelow` depth. When a collapsed folder is encountered, set `hideBelow` to its depth. Skip nodes deeper than `hideBelow`. When a node at `<= hideBelow` depth is encountered, reset.

### toggleFolder() changes

Match by `Collection + Path` instead of just `Collection` name. Top-level collection folders match with `Path == ""`.

### No changes to

- `domain/` — Collection struct stays flat with `[]Request`
- `storage/` — JSON format unchanged
- `reqeditor/`, `main.go` — unaffected
- Import script — already encodes paths as ` / ` in names
