# RataTUIle Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. REQUIRED: Follow go-expertise skill for all Go code.

**Goal:** Build a keyboard-driven TUI API client in Go — a personal Postman replacement.

**Architecture:** Three layers — TUI (bubbletea v2 + lipgloss v2 + huh), Core (net/http, env manager), Storage (JSON file I/O). All app packages under `internal/`.

**Tech Stack:** Go, bubbletea v2 (`charm.land/bubbletea/v2`), lipgloss v2 (`charm.land/lipgloss/v2`), huh, bubbles, net/http

**IMPORTANT API NOTE:** Bubbletea v2 uses `tea.KeyPressMsg` (not `tea.KeyMsg`), `tea.View` return type from `View()`, and `tea.NewView()` to construct views.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/ratatuile/main.go`
- Create: `Makefile`

**Step 1: Initialize Go module**

```bash
cd /Users/erlandas/Documents/personal/ratatuile
go mod init github.com/erlandas/ratatuile
```

**Step 2: Create entry point**

Create `cmd/ratatuile/main.go`:

```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

// Why: model is the root application state for bubbletea's MVU architecture.
// All child components will be fields on this struct.
type model struct{}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView("ratatuile - press q to quit\n")
}

func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Create Makefile**

```makefile
.PHONY: build test lint run

build:
	go build -o bin/ratatuile ./cmd/ratatuile

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run

run:
	go run ./cmd/ratatuile
```

**Step 4: Install dependencies and verify**

```bash
go mod tidy
make run
```

Expected: TUI launches, shows text, q quits.

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: scaffold project with bubbletea v2 skeleton"
```

---

### Task 2: Domain Types

**Files:**
- Create: `internal/domain/request.go`
- Create: `internal/domain/environment.go`
- Create: `internal/domain/collection.go`
- Create: `internal/domain/request_test.go`

**Step 1: Write failing test for request validation**

Create `internal/domain/request_test.go`:

```go
package domain

import "testing"

func TestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     Request
		wantErr bool
	}{
		{"valid GET", Request{Name: "test", Method: "GET", URL: "http://localhost"}, false},
		{"empty name", Request{Name: "", Method: "GET", URL: "http://localhost"}, true},
		{"empty URL", Request{Name: "test", Method: "GET", URL: ""}, true},
		{"empty method", Request{Name: "test", Method: "", URL: "http://localhost"}, true},
		{"valid POST", Request{Name: "test", Method: "POST", URL: "http://localhost", Body: `{"a":1}`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/domain/ -v
```

Expected: FAIL — `Request` type not defined.

**Step 3: Implement domain types**

Create `internal/domain/request.go`:

```go
package domain

import "fmt"

// Request represents a saved HTTP request.
type Request struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
	Body    string            `json:"body,omitempty"`
}

func (r Request) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("request name must not be empty")
	}
	if r.Method == "" {
		return fmt.Errorf("request method must not be empty")
	}
	if r.URL == "" {
		return fmt.Errorf("request URL must not be empty")
	}
	return nil
}
```

Create `internal/domain/environment.go`:

```go
package domain

// Environment holds named variables for template substitution.
type Environment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}
```

Create `internal/domain/collection.go`:

```go
package domain

// Collection groups requests under a name.
type Collection struct {
	Name     string    `json:"name"`
	Requests []Request `json:"requests,omitempty"`
}
```

**Step 4: Run tests**

```bash
go test ./internal/domain/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/ && git commit -m "feat: add domain types with request validation"
```

---

### Task 3: Storage Layer

**Files:**
- Create: `internal/storage/store.go`
- Create: `internal/storage/store_test.go`

**Step 1: Write failing tests**

Create `internal/storage/store_test.go`:

```go
package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/erlandas/ratatuile/internal/domain"
)

func TestStore_SaveAndLoadCollection(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	ctx := context.Background()

	col := domain.Collection{
		Name: "test-api",
		Requests: []domain.Request{
			{Name: "Get Users", Method: "GET", URL: "http://localhost/users"},
		},
	}

	if err := s.SaveCollection(ctx, col); err != nil {
		t.Fatalf("SaveCollection: %v", err)
	}

	got, err := s.LoadCollection(ctx, "test-api")
	if err != nil {
		t.Fatalf("LoadCollection: %v", err)
	}
	if got.Name != col.Name {
		t.Errorf("name = %q, want %q", got.Name, col.Name)
	}
	if len(got.Requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(got.Requests))
	}
}

func TestStore_ListCollections(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	ctx := context.Background()

	for _, name := range []string{"api-a", "api-b"} {
		if err := s.SaveCollection(ctx, domain.Collection{Name: name}); err != nil {
			t.Fatalf("SaveCollection(%s): %v", name, err)
		}
	}

	names, err := s.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("len = %d, want 2", len(names))
	}
}

func TestStore_SaveAndLoadEnvironment(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)
	ctx := context.Background()

	env := domain.Environment{
		Name:      "dev",
		Variables: map[string]string{"base_url": "http://localhost:3000"},
	}

	if err := s.SaveEnvironment(ctx, env); err != nil {
		t.Fatalf("SaveEnvironment: %v", err)
	}

	got, err := s.LoadEnvironment(ctx, "dev")
	if err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}
	if got.Variables["base_url"] != "http://localhost:3000" {
		t.Errorf("base_url = %q, want http://localhost:3000", got.Variables["base_url"])
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/storage/ -v
```

Expected: FAIL — package doesn't exist.

**Step 3: Implement storage**

Create `internal/storage/store.go`:

```go
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/erlandas/ratatuile/internal/domain"
)

// Why: FileStore takes a root dir via constructor — dependency injection,
// not a hardcoded path. Tests use t.TempDir().
type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (s *FileStore) collectionsDir() string {
	return filepath.Join(s.root, "collections")
}

func (s *FileStore) environmentsDir() string {
	return filepath.Join(s.root, "environments")
}

func (s *FileStore) SaveCollection(_ context.Context, col domain.Collection) error {
	dir := filepath.Join(s.collectionsDir(), col.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating collection dir %q: %w", col.Name, err)
	}
	return writeJSON(filepath.Join(dir, "collection.json"), col)
}

func (s *FileStore) LoadCollection(_ context.Context, name string) (domain.Collection, error) {
	path := filepath.Join(s.collectionsDir(), name, "collection.json")
	var col domain.Collection
	if err := readJSON(path, &col); err != nil {
		return col, fmt.Errorf("loading collection %q: %w", name, err)
	}
	return col, nil
}

func (s *FileStore) ListCollections(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.collectionsDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("listing collections: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func (s *FileStore) SaveEnvironment(_ context.Context, env domain.Environment) error {
	dir := s.environmentsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating environments dir: %w", err)
	}
	return writeJSON(filepath.Join(dir, env.Name+".json"), env)
}

func (s *FileStore) LoadEnvironment(_ context.Context, name string) (domain.Environment, error) {
	path := filepath.Join(s.environmentsDir(), name+".json")
	var env domain.Environment
	if err := readJSON(path, &env); err != nil {
		return env, fmt.Errorf("loading environment %q: %w", name, err)
	}
	return env, nil
}

func (s *FileStore) ListEnvironments(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.environmentsDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	return nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/storage/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/storage/ && git commit -m "feat: add JSON file storage layer"
```

---

### Task 4: HTTP Client

**Files:**
- Create: `internal/httpclient/client.go`
- Create: `internal/httpclient/client_test.go`

**Step 1: Write failing test**

Create `internal/httpclient/client_test.go`:

```go
package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erlandas/ratatuile/internal/domain"
)

func TestClient_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "hello" {
			t.Errorf("missing custom header")
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("missing query param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient()
	req := domain.Request{
		Name:    "test",
		Method:  "GET",
		URL:     srv.URL,
		Headers: map[string]string{"X-Custom": "hello"},
		Params:  map[string]string{"page": "1"},
	}

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Body == "" {
		t.Error("body is empty")
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/httpclient/ -v
```

Expected: FAIL — package doesn't exist.

**Step 3: Implement HTTP client**

Create `internal/httpclient/client.go`:

```go
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/erlandas/ratatuile/internal/domain"
)

// Response holds the result of an HTTP request execution.
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	Size       int               `json:"size"`
}

// Why: Client wraps http.Client so we can inject timeouts and
// swap the transport in tests if needed.
type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// Do executes a domain.Request and returns a Response.
// Why: context.Context first param — caller controls cancellation/timeouts.
func (c *Client) Do(ctx context.Context, req domain.Request) (Response, error) {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return Response{}, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	q := httpReq.URL.Query()
	for k, v := range req.Params {
		q.Set(k, v)
	}
	httpReq.URL.RawQuery = q.Encode()

	start := time.Now()
	resp, err := c.http.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return Response{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading response body: %w", err)
	}

	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	return Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
		Duration:   duration,
		Size:       len(body),
	}, nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/httpclient/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/httpclient/ && git commit -m "feat: add HTTP client with context support"
```

---

### Task 5: Variable Substitution

**Files:**
- Create: `internal/envmanager/envmanager.go`
- Create: `internal/envmanager/envmanager_test.go`

**Step 1: Write failing test**

Create `internal/envmanager/envmanager_test.go`:

```go
package envmanager

import (
	"testing"

	"github.com/erlandas/ratatuile/internal/domain"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name string
		input string
		vars  map[string]string
		want  string
	}{
		{"simple", "{{base_url}}/users", map[string]string{"base_url": "http://localhost"}, "http://localhost/users"},
		{"multiple", "{{host}}:{{port}}", map[string]string{"host": "localhost", "port": "8080"}, "localhost:8080"},
		{"no vars", "http://example.com", nil, "http://example.com"},
		{"missing var", "{{missing}}/path", map[string]string{}, "{{missing}}/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.input, tt.vars)
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveRequest(t *testing.T) {
	env := domain.Environment{
		Name:      "dev",
		Variables: map[string]string{"base": "http://localhost", "tok": "abc"},
	}
	req := domain.Request{
		Name:    "test",
		Method:  "GET",
		URL:     "{{base}}/users",
		Headers: map[string]string{"Authorization": "Bearer {{tok}}"},
	}

	got := ResolveRequest(req, env)
	if got.URL != "http://localhost/users" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Headers["Authorization"] != "Bearer abc" {
		t.Errorf("Authorization = %q", got.Headers["Authorization"])
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/envmanager/ -v
```

Expected: FAIL

**Step 3: Implement**

Create `internal/envmanager/envmanager.go`:

```go
package envmanager

import (
	"strings"

	"github.com/erlandas/ratatuile/internal/domain"
)

// Resolve replaces {{key}} placeholders with values from vars.
// Unresolved placeholders are left as-is.
func Resolve(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

// ResolveRequest returns a copy of req with all {{var}} placeholders
// replaced using the environment's variables.
func ResolveRequest(req domain.Request, env domain.Environment) domain.Request {
	resolved := req
	resolved.URL = Resolve(req.URL, env.Variables)
	resolved.Body = Resolve(req.Body, env.Variables)

	if len(req.Headers) > 0 {
		resolved.Headers = make(map[string]string, len(req.Headers))
		for k, v := range req.Headers {
			resolved.Headers[k] = Resolve(v, env.Variables)
		}
	}

	if len(req.Params) > 0 {
		resolved.Params = make(map[string]string, len(req.Params))
		for k, v := range req.Params {
			resolved.Params[k] = Resolve(v, env.Variables)
		}
	}

	return resolved
}
```

**Step 4: Run tests**

```bash
go test ./internal/envmanager/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/envmanager/ && git commit -m "feat: add variable substitution engine"
```

---

### Task 6: Collection Tree TUI Component

**Files:**
- Create: `internal/tui/tree/tree.go`

This is the left pane — a navigable tree of collections and their requests.

**Step 1: Implement tree component**

Create `internal/tui/tree/tree.go`:

```go
package tree

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/domain"
)

// Why: each node is either a collection (folder) or a request (leaf).
// Keeping them in a flat list with depth simplifies cursor navigation.
type Node struct {
	Name       string
	IsFolder   bool
	Expanded   bool
	Depth      int
	Collection string
	Request    *domain.Request
}

type Model struct {
	nodes    []Node
	cursor   int
	focused  bool
	width    int
	height   int
}

// Why: messages for parent to know what was selected
type RequestSelectedMsg struct {
	Collection string
	Request    domain.Request
}

func New(collections []domain.Collection) Model {
	var nodes []Node
	for _, col := range collections {
		nodes = append(nodes, Node{
			Name:       col.Name,
			IsFolder:   true,
			Depth:      0,
			Collection: col.Name,
		})
		for i := range col.Requests {
			nodes = append(nodes, Node{
				Name:       col.Requests[i].Name,
				IsFolder:   false,
				Depth:      1,
				Collection: col.Name,
				Request:    &col.Requests[i],
			})
		}
	}
	return Model{nodes: nodes}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) { m.focused = f }
func (m Model) Focused() bool      { return m.focused }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		visible := m.visibleNodes()
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor < len(visible) {
				node := visible[m.cursor]
				if node.IsFolder {
					m.toggleFolder(node.Collection)
				} else if node.Request != nil {
					return m, func() tea.Msg {
						return RequestSelectedMsg{
							Collection: node.Collection,
							Request:    *node.Request,
						}
					}
				}
			}
		}
	}
	return m, nil
}

func (m *Model) toggleFolder(name string) {
	for i := range m.nodes {
		if m.nodes[i].IsFolder && m.nodes[i].Collection == name {
			m.nodes[i].Expanded = !m.nodes[i].Expanded
			break
		}
	}
}

// Why: visibleNodes filters to only show requests under expanded folders.
func (m Model) visibleNodes() []Node {
	var visible []Node
	showChildren := false
	for _, n := range m.nodes {
		if n.IsFolder {
			visible = append(visible, n)
			showChildren = n.Expanded
		} else if showChildren {
			visible = append(visible, n)
		}
	}
	return visible
}

func (m Model) View() string {
	visible := m.visibleNodes()

	// Why: lipgloss styles for focused vs unfocused states
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var s string
	s += titleStyle.Render("Collections") + "\n\n"

	for i, node := range visible {
		prefix := "  "
		if m.focused && i == m.cursor {
			prefix = cursorStyle.Render("> ")
		}

		indent := ""
		for j := 0; j < node.Depth; j++ {
			indent += "  "
		}

		label := node.Name
		if node.IsFolder {
			arrow := ">"
			if node.Expanded {
				arrow = "v"
			}
			label = fmt.Sprintf("%s %s", arrow, node.Name)
		} else if node.Request != nil {
			method := dimStyle.Render(node.Request.Method)
			label = fmt.Sprintf("%s %s", method, node.Name)
		}

		s += prefix + indent + label + "\n"
	}

	return s
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/tui/tree/
```

Expected: compiles without error.

**Step 3: Commit**

```bash
git add internal/tui/tree/ && git commit -m "feat: add collection tree TUI component"
```

---

### Task 7: Request Editor TUI Component

**Files:**
- Create: `internal/tui/reqeditor/reqeditor.go`

The top-right pane — displays method, URL, and tabbed sections for headers/params/body. Uses huh forms for editing.

**Step 1: Implement request editor**

Create `internal/tui/reqeditor/reqeditor.go`:

```go
package reqeditor

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/domain"
)

type Tab int

const (
	TabHeaders Tab = iota
	TabParams
	TabBody
)

func (t Tab) String() string {
	switch t {
	case TabHeaders:
		return "Headers"
	case TabParams:
		return "Params"
	case TabBody:
		return "Body"
	}
	return ""
}

// Why: SendRequestMsg tells the parent to execute this request.
type SendRequestMsg struct{ Request domain.Request }

type Model struct {
	request  domain.Request
	activeTab Tab
	focused  bool
	width    int
	height   int
}

func New() Model {
	return Model{}
}

func (m *Model) SetRequest(r domain.Request) { m.request = r }
func (m *Model) SetSize(w, h int)            { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool)           { m.focused = f }
func (m Model) Focused() bool                { return m.focused }
func (m Model) Request() domain.Request      { return m.request }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
		case "shift+tab":
			m.activeTab = (m.activeTab + 2) % 3
		case "ctrl+s":
			return m, func() tea.Msg {
				return SendRequestMsg{Request: m.request}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	activeTabStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Underline(true)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	methodStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))

	var s string

	// Method + URL line
	method := m.request.Method
	if method == "" {
		method = "GET"
	}
	s += titleStyle.Render("Request") + "\n"
	s += methodStyle.Render(method) + " " + m.request.URL + "\n\n"

	// Tab bar
	tabs := []Tab{TabHeaders, TabParams, TabBody}
	var tabLine []string
	for _, t := range tabs {
		if t == m.activeTab {
			tabLine = append(tabLine, activeTabStyle.Render(t.String()))
		} else {
			tabLine = append(tabLine, inactiveTabStyle.Render(t.String()))
		}
	}
	s += strings.Join(tabLine, "  |  ") + "\n\n"

	// Tab content
	switch m.activeTab {
	case TabHeaders:
		s += m.renderMap(m.request.Headers)
	case TabParams:
		s += m.renderMap(m.request.Params)
	case TabBody:
		if m.request.Body != "" {
			s += m.request.Body
		} else {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(no body)")
		}
	}

	return s
}

func (m Model) renderMap(kv map[string]string) string {
	if len(kv) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(empty)")
	}
	var lines []string
	for k, v := range kv {
		lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
	}
	return strings.Join(lines, "\n")
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/tui/reqeditor/
```

**Step 3: Commit**

```bash
git add internal/tui/reqeditor/ && git commit -m "feat: add request editor TUI component"
```

---

### Task 8: Response Viewer TUI Component

**Files:**
- Create: `internal/tui/respview/respview.go`

The bottom-right pane — shows status, timing, headers, and syntax-highlighted JSON body.

**Step 1: Implement response viewer**

Create `internal/tui/respview/respview.go`:

```go
package respview

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/httpclient"
)

type Model struct {
	response *httpclient.Response
	focused  bool
	scroll   int
	width    int
	height   int
}

func New() Model { return Model{} }

func (m *Model) SetResponse(r *httpclient.Response) {
	m.response = r
	m.scroll = 0
}
func (m *Model) SetSize(w, h int)  { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool) { m.focused = f }
func (m Model) Focused() bool      { return m.focused }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	s := titleStyle.Render("Response") + "\n"

	if m.response == nil {
		s += dimStyle.Render("Send a request with Ctrl+S")
		return s
	}

	r := m.response

	// Status line with color coding
	statusStyle := lipgloss.NewStyle().Bold(true)
	switch {
	case r.StatusCode >= 200 && r.StatusCode < 300:
		statusStyle = statusStyle.Foreground(lipgloss.Color("86"))
	case r.StatusCode >= 300 && r.StatusCode < 400:
		statusStyle = statusStyle.Foreground(lipgloss.Color("228"))
	default:
		statusStyle = statusStyle.Foreground(lipgloss.Color("196"))
	}

	s += statusStyle.Render(fmt.Sprintf("%d", r.StatusCode))
	s += dimStyle.Render(fmt.Sprintf("  %s  %d bytes", r.Duration.Round(1e6), r.Size))
	s += "\n\n"

	// Headers
	s += dimStyle.Render("Headers:") + "\n"
	for k, v := range r.Headers {
		s += fmt.Sprintf("  %s: %s\n", k, v)
	}
	s += "\n"

	// Body — try to pretty-print JSON
	body := r.Body
	var js json.RawMessage
	if json.Unmarshal([]byte(body), &js) == nil {
		if pretty, err := json.MarshalIndent(js, "", "  "); err == nil {
			body = string(pretty)
		}
	}

	lines := strings.Split(body, "\n")
	// Why: simple scroll by skipping lines. Clamp to avoid overscroll.
	if m.scroll > len(lines) {
		m.scroll = len(lines)
	}
	visible := lines[m.scroll:]
	if m.height > 0 && len(visible) > m.height-8 {
		visible = visible[:m.height-8]
	}

	s += strings.Join(visible, "\n")

	return s
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/tui/respview/
```

**Step 3: Commit**

```bash
git add internal/tui/respview/ && git commit -m "feat: add response viewer TUI component"
```

---

### Task 9: Root Model — Wire Three-Pane Layout

**Files:**
- Modify: `cmd/ratatuile/main.go` (replace skeleton)

Wire the tree, request editor, and response viewer into a three-pane layout with focus cycling (Ctrl+W) and request execution (Ctrl+S).

**Step 1: Rewrite main.go with full root model**

Replace `cmd/ratatuile/main.go` with:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/domain"
	"github.com/erlandas/ratatuile/internal/envmanager"
	"github.com/erlandas/ratatuile/internal/httpclient"
	"github.com/erlandas/ratatuile/internal/storage"
	"github.com/erlandas/ratatuile/internal/tui/reqeditor"
	"github.com/erlandas/ratatuile/internal/tui/respview"
	"github.com/erlandas/ratatuile/internal/tui/tree"
)

// Why: Pane enum for focus management. Cycling through panes with Ctrl+W.
type pane int

const (
	paneTree pane = iota
	paneRequest
	paneResponse
	paneCount
)

type model struct {
	tree      tree.Model
	reqEditor reqeditor.Model
	respView  respview.Model

	store      *storage.FileStore
	client     *httpclient.Client
	activeEnv  *domain.Environment
	focusedPane pane
	width      int
	height     int
	err        error
}

// Why: responseMsg wraps the async HTTP result so bubbletea can deliver it.
type responseMsg struct {
	resp *httpclient.Response
	err  error
}

func initialModel(store *storage.FileStore) model {
	ctx := context.Background()
	names, _ := store.ListCollections(ctx)
	var collections []domain.Collection
	for _, n := range names {
		col, err := store.LoadCollection(ctx, n)
		if err == nil {
			collections = append(collections, col)
		}
	}

	m := model{
		tree:      tree.New(collections),
		reqEditor: reqeditor.New(),
		respView:  respview.New(),
		store:     store,
		client:    httpclient.NewClient(),
	}
	m.tree.SetFocused(true)
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutPanes()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+w":
			m.cycleFocus()
			return m, nil
		}

	case tree.RequestSelectedMsg:
		m.reqEditor.SetRequest(msg.Request)
		return m, nil

	case reqeditor.SendRequestMsg:
		req := msg.Request
		if m.activeEnv != nil {
			req = envmanager.ResolveRequest(req, *m.activeEnv)
		}
		return m, m.executeRequest(req)

	case responseMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.respView.SetResponse(msg.resp)
			m.err = nil
		}
		return m, nil
	}

	// Delegate to focused child
	var cmd tea.Cmd
	switch m.focusedPane {
	case paneTree:
		m.tree, cmd = m.tree.Update(msg)
	case paneRequest:
		m.reqEditor, cmd = m.reqEditor.Update(msg)
	case paneResponse:
		m.respView, cmd = m.respView.Update(msg)
	}
	return m, cmd
}

func (m *model) cycleFocus() {
	m.tree.SetFocused(false)
	m.reqEditor.SetFocused(false)
	m.respView.SetFocused(false)

	m.focusedPane = (m.focusedPane + 1) % paneCount

	switch m.focusedPane {
	case paneTree:
		m.tree.SetFocused(true)
	case paneRequest:
		m.reqEditor.SetFocused(true)
	case paneResponse:
		m.respView.SetFocused(true)
	}
}

func (m *model) layoutPanes() {
	leftW := m.width * 30 / 100
	rightW := m.width - leftW - 1
	topH := m.height * 40 / 100
	bottomH := m.height - topH - 1

	m.tree.SetSize(leftW, m.height)
	m.reqEditor.SetSize(rightW, topH)
	m.respView.SetSize(rightW, bottomH)
}

func (m model) executeRequest(req domain.Request) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		resp, err := client.Do(context.Background(), req)
		if err != nil {
			return responseMsg{err: err}
		}
		return responseMsg{resp: &resp}
	}
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	leftW := m.width * 30 / 100
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	focusBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86"))

	treeStyle := borderStyle
	reqStyle := borderStyle
	respStyle := borderStyle
	switch m.focusedPane {
	case paneTree:
		treeStyle = focusBorderStyle
	case paneRequest:
		reqStyle = focusBorderStyle
	case paneResponse:
		respStyle = focusBorderStyle
	}

	leftPane := treeStyle.Width(leftW - 2).Height(m.height - 2).Render(m.tree.View())
	rightW := m.width - leftW - 1
	topH := m.height * 40 / 100
	bottomH := m.height - topH - 1

	topPane := reqStyle.Width(rightW - 2).Height(topH - 2).Render(m.reqEditor.View())
	bottomPane := respStyle.Width(rightW - 2).Height(bottomH - 2).Render(m.respView.View())

	// Error bar
	errLine := ""
	if m.err != nil {
		errLine = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err))
	}
	_ = errLine

	rightSide := lipgloss.JoinVertical(lipgloss.Left, topPane, bottomPane)
	layout := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightSide)

	return tea.NewView(layout)
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	store := storage.NewFileStore(filepath.Join(home, ".ratatuile"))
	p := tea.NewProgram(initialModel(store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./cmd/ratatuile/
```

**Step 3: Manual test**

```bash
make run
```

Expected: Three-pane layout renders. Ctrl+W cycles focus (border color changes). q quits.

**Step 4: Commit**

```bash
git add cmd/ratatuile/main.go && git commit -m "feat: wire three-pane layout with focus cycling"
```

---

### Task 10: Environment Switching (Ctrl+E)

**Files:**
- Create: `internal/tui/envpicker/envpicker.go`
- Modify: `cmd/ratatuile/main.go`

An overlay that lists environments and lets the user pick one with j/k + Enter. Ctrl+E toggles it.

**Step 1: Implement environment picker**

Create `internal/tui/envpicker/envpicker.go`:

```go
package envpicker

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/domain"
)

type EnvSelectedMsg struct{ Env domain.Environment }
type DismissMsg struct{}

type Model struct {
	envs    []domain.Environment
	cursor  int
	visible bool
}

func New(envs []domain.Environment) Model {
	return Model{envs: envs}
}

func (m *Model) Toggle()          { m.visible = !m.visible }
func (m Model) Visible() bool     { return m.visible }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.envs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor < len(m.envs) {
				env := m.envs[m.cursor]
				m.visible = false
				return m, func() tea.Msg { return EnvSelectedMsg{Env: env} }
			}
		case "escape", "ctrl+e":
			m.visible = false
			return m, func() tea.Msg { return DismissMsg{} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	s := title.Render("Switch Environment") + "\n\n"
	for i, env := range m.envs {
		prefix := "  "
		if i == m.cursor {
			prefix = cursor.Render("> ")
		}
		s += prefix + env.Name + "\n"
	}
	s += "\nEnter to select, Esc to cancel"

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
```

**Step 2: Wire into root model**

Add to `cmd/ratatuile/main.go`:
- Import `envpicker` package
- Add `envPicker envpicker.Model` field to `model` struct
- In `initialModel`, load environments and create picker
- Handle `ctrl+e` in Update to toggle picker
- Handle `envpicker.EnvSelectedMsg` to set `m.activeEnv`
- When picker is visible, delegate all input to it
- Overlay picker view on top of layout in View()

**Step 3: Verify it compiles and test manually**

```bash
go build ./cmd/ratatuile/ && make run
```

Expected: Ctrl+E shows overlay, j/k navigates, Enter selects, Esc dismisses.

**Step 4: Commit**

```bash
git add internal/tui/envpicker/ cmd/ratatuile/main.go && git commit -m "feat: add environment switching with Ctrl+E"
```

---

### Task 11: Help Overlay

**Files:**
- Create: `internal/tui/help/help.go`
- Modify: `cmd/ratatuile/main.go`

Toggled with `?`. Shows keybinding reference.

**Step 1: Implement help overlay**

Create `internal/tui/help/help.go`:

```go
package help

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type DismissMsg struct{}

type Model struct {
	visible bool
}

func New() Model { return Model{} }

func (m *Model) Toggle()      { m.visible = !m.visible }
func (m Model) Visible() bool { return m.visible }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "?", "escape":
			m.visible = false
			return m, func() tea.Msg { return DismissMsg{} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	key := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	s := title.Render("Keybindings") + "\n\n"
	bindings := [][2]string{
		{"Ctrl+W", "Cycle pane focus"},
		{"j/k", "Navigate up/down"},
		{"Enter", "Expand/select"},
		{"Tab", "Switch editor tab"},
		{"Ctrl+S", "Send request"},
		{"Ctrl+E", "Switch environment"},
		{"?", "Toggle this help"},
		{"q / Ctrl+C", "Quit"},
	}
	for _, b := range bindings {
		s += key.Render(b[0]) + "  " + dim.Render(b[1]) + "\n"
	}
	s += "\nPress ? or Esc to close"

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
```

**Step 2: Wire into root model**

Add to `cmd/ratatuile/main.go`:
- Import `help` package
- Add `help help.Model` field
- Handle `?` key to toggle help
- When help is visible, delegate input to it
- Overlay help view centered on screen

**Step 3: Verify and test**

```bash
go build ./cmd/ratatuile/ && make run
```

Expected: `?` shows help overlay, `?` or Esc dismisses it.

**Step 4: Commit**

```bash
git add internal/tui/help/ cmd/ratatuile/main.go && git commit -m "feat: add help overlay with ? toggle"
```

---

### Task 12: Final Integration Test

**Step 1: Create sample data**

```bash
mkdir -p ~/.ratatuile/collections/sample-api
mkdir -p ~/.ratatuile/environments
```

Write `~/.ratatuile/collections/sample-api/collection.json`:
```json
{
  "name": "sample-api",
  "requests": [
    {
      "name": "Healthcheck",
      "method": "GET",
      "url": "{{base_url}}/health"
    },
    {
      "name": "Get Posts",
      "method": "GET",
      "url": "{{base_url}}/posts",
      "params": {"_limit": "5"}
    }
  ]
}
```

Write `~/.ratatuile/environments/jsonplaceholder.json`:
```json
{
  "name": "jsonplaceholder",
  "variables": {
    "base_url": "https://jsonplaceholder.typicode.com"
  }
}
```

**Step 2: Run full app**

```bash
make run
```

**Verify:**
1. Collection tree shows "sample-api" with 2 requests
2. Enter expands collection, Enter on request loads it in editor
3. Ctrl+E switches to jsonplaceholder environment
4. Ctrl+S sends request, response shows in bottom-right
5. Ctrl+W cycles focus between panes
6. ? shows help overlay

**Step 3: Final commit**

```bash
git add -A && git commit -m "feat: complete MVP ratatuile TUI"
```