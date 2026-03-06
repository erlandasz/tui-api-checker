# Sub-Collection Tree Display Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Parse ` / ` separators in request names to render nested expandable/collapsible sub-folders in the tree component.

**Architecture:** Add a `Path` field to `Node` for folder identity. Refactor `New()` to build intermediate folder nodes from request name prefixes. Replace the flat `visibleNodes()` with a depth-aware algorithm. Only `internal/tui/tree/tree.go` is modified.

**Tech Stack:** Go, Bubble Tea v2

---

### Task 1: Add Path field to Node and refactor toggleFolder

**Files:**
- Modify: `internal/tui/tree/tree.go:13-20` (Node struct)
- Modify: `internal/tui/tree/tree.go:88-92` (Update toggle call)
- Modify: `internal/tui/tree/tree.go:115-122` (toggleFolder)

**Step 1: Add `Path` field to Node struct**

```go
type Node struct {
	Name       string
	IsFolder   bool
	Expanded   bool
	Depth      int
	Collection string
	Path       string // folder path, e.g. "Auth/Login". Empty for top-level collection folders.
	Request    *domain.Request
}
```

**Step 2: Update toggleFolder to match by Collection + Path**

```go
func (m *Model) toggleFolder(collection, path string) {
	for i := range m.nodes {
		if m.nodes[i].IsFolder && m.nodes[i].Collection == collection && m.nodes[i].Path == path {
			m.nodes[i].Expanded = !m.nodes[i].Expanded
			break
		}
	}
}
```

**Step 3: Update the call site in Update()**

Change line 92 from:
```go
m.toggleFolder(node.Collection)
```
to:
```go
m.toggleFolder(node.Collection, node.Path)
```

**Step 4: Run `make build` — expected: compiles OK**

**Step 5: Commit** `refactor(tree): add Path field to Node and update toggleFolder`

---

### Task 2: Refactor New() to build sub-folder nodes

**Files:**
- Modify: `internal/tui/tree/tree.go:40-60` (New function)

**Step 1: Add `strings` import**

**Step 2: Replace the New() function body**

```go
func New(collections []domain.Collection) Model {
	var nodes []Node
	for _, col := range collections {
		nodes = append(nodes, Node{
			Name:       col.Name,
			IsFolder:   true,
			Depth:      0,
			Collection: col.Name,
			Path:       "",
		})

		// Track which sub-folder paths we've already created
		seen := make(map[string]bool)

		for i := range col.Requests {
			req := &col.Requests[i]
			parts := strings.Split(req.Name, " / ")

			// Create intermediate folder nodes for each prefix
			for j := 0; j < len(parts)-1; j++ {
				folderPath := strings.Join(parts[:j+1], "/")
				if seen[folderPath] {
					continue
				}
				seen[folderPath] = true
				nodes = append(nodes, Node{
					Name:       parts[j],
					IsFolder:   true,
					Depth:      j + 1,
					Collection: col.Name,
					Path:       folderPath,
				})
			}

			// Add the request leaf node with just the final name segment
			nodes = append(nodes, Node{
				Name:       parts[len(parts)-1],
				IsFolder:   false,
				Depth:      len(parts),
				Collection: col.Name,
				Path:       strings.Join(parts[:len(parts)-1], "/"),
				Request:    req,
			})
		}
	}
	return Model{nodes: nodes}
}
```

Key details:
- `parts` splits `"Auth / Login / Get Token"` into `["Auth", "Login", "Get Token"]`
- For each prefix segment, creates a folder node if not already `seen`
- Request leaf gets `Depth = len(parts)` (1 for no-folder requests, deeper for nested)
- Request leaf `Path` is the parent folder path (empty string if no folders)
- `seen` map prevents duplicate folder nodes when multiple requests share a prefix

**Step 3: Run `make build` — expected: compiles OK**

**Step 4: Commit** `feat(tree): build sub-folder nodes from request name prefixes`

---

### Task 3: Refactor visibleNodes() for depth-aware visibility

**Files:**
- Modify: `internal/tui/tree/tree.go:125-137` (visibleNodes)

**Step 1: Replace visibleNodes()**

```go
func (m Model) visibleNodes() []Node {
	var visible []Node
	hideBelow := -1 // -1 means nothing hidden
	for _, n := range m.nodes {
		// If we're hiding and this node is deeper than the collapsed folder, skip it
		if hideBelow >= 0 && n.Depth > hideBelow {
			continue
		}
		// We've reached a node at same or lesser depth — stop hiding
		hideBelow = -1

		visible = append(visible, n)

		// If this is a collapsed folder, hide everything deeper
		if n.IsFolder && !n.Expanded {
			hideBelow = n.Depth
		}
	}
	return visible
}
```

Logic: when a collapsed folder is encountered at depth D, set `hideBelow = D`. All subsequent nodes with depth > D are skipped. When a node at depth <= D is found, reset `hideBelow`.

**Step 2: Run `make build` — expected: compiles OK**

**Step 3: Run `make run`, test: expand a collection with ` / ` named requests, verify sub-folders appear and toggle**

**Step 4: Commit** `feat(tree): depth-aware visibility for sub-folder collapse`

---

### Task 4: Update AddRequest for depth-aware insertion

**Files:**
- Modify: `internal/tui/tree/tree.go:139-178` (AddRequest)

**Step 1: Update AddRequest**

The existing `AddRequest` hardcodes `Depth: 1`. New requests created via `n` don't have ` / ` in their name, so they should insert at depth 1 with empty Path — but we need to find the right insertion point which is now after all sub-folder nodes too.

```go
func (m *Model) AddRequest(collection string, req domain.Request) {
	insertIdx := -1
	for i, n := range m.nodes {
		if n.Collection == collection {
			insertIdx = i + 1
			// Expand the top-level collection folder
			if n.IsFolder && n.Path == "" {
				m.nodes[i].Expanded = true
			}
		} else if insertIdx >= 0 {
			break
		}
	}
	if insertIdx < 0 {
		return
	}

	newNode := Node{
		Name:       req.Name,
		IsFolder:   false,
		Depth:      1,
		Collection: collection,
		Path:       "",
		Request:    &req,
	}

	m.nodes = append(m.nodes, Node{})
	copy(m.nodes[insertIdx+1:], m.nodes[insertIdx:])
	m.nodes[insertIdx] = newNode

	for i, n := range m.visibleNodes() {
		if !n.IsFolder && n.Request != nil && n.Request.Name == req.Name && n.Collection == collection {
			m.cursor = i
			break
		}
	}
}
```

**Step 2: Run `make build` and `make test` — expected: pass**

**Step 3: Commit** `fix(tree): depth-aware AddRequest insertion`

---

### Task 5: Build, lint, verify

**Step 1:** Run `make build && make lint && make test` — all pass
**Step 2:** Run `make run`, test with real imported collections that have ` / ` names
**Step 3:** Fix any issues found
**Step 4: Commit** any fixes
