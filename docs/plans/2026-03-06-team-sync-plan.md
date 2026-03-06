# Team Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let teams push/pull encrypted collections to a central Cloudflare Worker + KV store using a shared group passphrase.

**Architecture:** New `internal/sync/` package handles crypto (AES-256-GCM) and HTTP client calls. New `internal/tui/syncmodal/` provides the TUI overlay. Settings (group key, endpoint) persisted via `FileStore`. Cloudflare Worker deployed separately as `worker/sync/`.

**Tech Stack:** Go stdlib `crypto/aes`, `crypto/cipher`, `crypto/sha256`, `golang.org/x/crypto/scrypt`, `net/http`. Cloudflare Worker in JS.

---

### Task 1: Crypto — key derivation, encrypt, decrypt

**Files:**
- Create: `internal/sync/crypto.go`
- Create: `internal/sync/crypto_test.go`

**Step 1: Write failing tests**

```go
// internal/sync/crypto_test.go
package sync

import (
	"testing"
)

func TestDeriveKey(t *testing.T) {
	ns1, key1 := DeriveKey("my-team-secret")
	ns2, key2 := DeriveKey("my-team-secret")

	if ns1 != ns2 {
		t.Fatalf("namespace not deterministic: %s != %s", ns1, ns2)
	}
	if len(ns1) != 16 {
		t.Fatalf("namespace should be 16 hex chars, got %d", len(ns1))
	}
	if len(key1) != 32 {
		t.Fatalf("key should be 32 bytes, got %d", len(key1))
	}
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("key not deterministic")
		}
	}

	ns3, _ := DeriveKey("different-secret")
	if ns1 == ns3 {
		t.Fatal("different passphrases should produce different namespaces")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	_, key := DeriveKey("test-pass")
	plaintext := []byte(`{"name":"demo","requests":[]}`)

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip failed: got %q", decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	_, key1 := DeriveKey("key-one")
	_, key2 := DeriveKey("key-two")

	ciphertext, _ := Encrypt(key1, []byte("secret"))
	_, err := Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/sync/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write implementation**

```go
// internal/sync/crypto.go
package sync

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

func DeriveKey(passphrase string) (namespace string, aesKey []byte) {
	hash := sha256.Sum256([]byte(passphrase))
	namespace = fmt.Sprintf("%x", hash[:8])

	key, err := scrypt.Key([]byte(passphrase), []byte("ratatuile"), 32768, 8, 1, 32)
	if err != nil {
		panic("scrypt failed: " + err.Error())
	}
	return namespace, key
}

func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
```

**Step 4: Add scrypt dependency and run tests**

Run: `go get golang.org/x/crypto && go test ./internal/sync/ -v`
Expected: PASS (all 3 tests)

**Step 5: Commit**

```bash
git add internal/sync/crypto.go internal/sync/crypto_test.go go.mod go.sum
git commit -m "feat(sync): add AES-256-GCM crypto with scrypt key derivation"
```

---

### Task 2: HTTP client — push, pull, list remote

**Files:**
- Create: `internal/sync/client.go`
- Create: `internal/sync/client_test.go`

**Step 1: Write failing tests**

```go
// internal/sync/client_test.go
package sync

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPushAndPull(t *testing.T) {
	store := map[string][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			store[key] = body
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			data, ok := store[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(data)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	blob := []byte("encrypted-data")

	if err := c.Push("ns123", "my-collection", blob); err != nil {
		t.Fatalf("push: %v", err)
	}

	got, err := c.Pull("ns123", "my-collection")
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if string(got) != string(blob) {
		t.Fatalf("pull got %q, want %q", got, blob)
	}
}

func TestPullNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Pull("ns", "nope")
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
}

func TestListRemote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"col-a", "col-b"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	names, err := c.ListRemote("ns123")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 2 || names[0] != "col-a" || names[1] != "col-b" {
		t.Fatalf("unexpected names: %v", names)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/sync/ -run TestPush -v && go test ./internal/sync/ -run TestList -v`
Expected: FAIL — `NewClient` undefined

**Step 3: Write implementation**

```go
// internal/sync/client.go
package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const DefaultEndpoint = "https://sync.ratatuile.dev"

type Client struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}
}

func (c *Client) Push(namespace, collection string, blob []byte) error {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, namespace, collection)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(blob))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push failed (%d): %s", resp.StatusCode, body)
	}
	return nil
}

func (c *Client) Pull(namespace, collection string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, namespace, collection)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("pull: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("collection %q not found on remote", collection)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pull failed (%d): %s", resp.StatusCode, body)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) ListRemote(namespace string) ([]string, error) {
	url := fmt.Sprintf("%s/%s", c.endpoint, namespace)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list failed (%d)", resp.StatusCode)
	}
	var names []string
	if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
		return nil, fmt.Errorf("parsing list: %w", err)
	}
	return names, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/sync/ -v`
Expected: PASS (all 6 tests — 3 crypto + 3 client)

**Step 5: Commit**

```bash
git add internal/sync/client.go internal/sync/client_test.go
git commit -m "feat(sync): add HTTP client for push/pull/list"
```

---

### Task 3: Settings persistence — group key and sync endpoint

**Files:**
- Modify: `internal/storage/store.go` (add 4 methods)
- Modify: `internal/storage/store_test.go` (add tests)

**Step 1: Write failing tests**

Add to `internal/storage/store_test.go`:

```go
func TestGroupKey(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	if got := s.LoadGroupKey(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}

	if err := s.SaveGroupKey("my-team-secret"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := s.LoadGroupKey(); got != "my-team-secret" {
		t.Fatalf("expected my-team-secret, got %q", got)
	}
}

func TestSyncEndpoint(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStore(dir)

	if got := s.LoadSyncEndpoint(); got != "https://sync.ratatuile.dev" {
		t.Fatalf("expected default endpoint, got %q", got)
	}

	if err := s.SaveSyncEndpoint("https://custom.example.com"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := s.LoadSyncEndpoint(); got != "https://custom.example.com" {
		t.Fatalf("expected custom, got %q", got)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/storage/ -run TestGroupKey -v && go test ./internal/storage/ -run TestSyncEndpoint -v`
Expected: FAIL — methods undefined

**Step 3: Add methods to `internal/storage/store.go`**

Add after `LoadActiveEnv()`:

```go
func (s *FileStore) groupKeyPath() string {
	return filepath.Join(s.root, "group_key")
}

func (s *FileStore) SaveGroupKey(key string) error {
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.groupKeyPath(), []byte(key), 0644)
}

func (s *FileStore) LoadGroupKey() string {
	data, err := os.ReadFile(s.groupKeyPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (s *FileStore) syncEndpointPath() string {
	return filepath.Join(s.root, "sync_endpoint")
}

func (s *FileStore) SaveSyncEndpoint(endpoint string) error {
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.syncEndpointPath(), []byte(endpoint), 0644)
}

func (s *FileStore) LoadSyncEndpoint() string {
	data, err := os.ReadFile(s.syncEndpointPath())
	if err != nil {
		return "https://sync.ratatuile.dev"
	}
	return strings.TrimSpace(string(data))
}
```

**Step 4: Run tests**

Run: `go test ./internal/storage/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/storage/store.go internal/storage/store_test.go
git commit -m "feat(storage): add group key and sync endpoint persistence"
```

---

### Task 4: Sync modal TUI component

**Files:**
- Create: `internal/tui/syncmodal/syncmodal.go`

This modal follows the same pattern as `envpicker`: Toggle/Visible, captures all input when visible, emits messages to parent.

**Screens:**
1. `screenKeyPrompt` — shown when no group key is set. Text input for passphrase.
2. `screenList` — two-column list. Left = local collections, right = remote. Cursor navigates both. `p` to push, `l` to pull.

**Messages emitted to parent:**
- `DismissMsg{}` — modal closed
- `PushMsg{Collection string}` — user wants to push this collection
- `PullMsg{Collection string}` — user wants to pull this collection
- `KeySavedMsg{Key string}` — user entered a group key

**Step 1: Create the component**

```go
// internal/tui/syncmodal/syncmodal.go
package syncmodal

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type DismissMsg struct{}
type PushMsg struct{ Collection string }
type PullMsg struct{ Collection string }
type KeySavedMsg struct{ Key string }

type screen int

const (
	screenKeyPrompt screen = iota
	screenList
)

type Model struct {
	visible bool
	screen  screen

	// Key prompt
	keyBuf    string
	keyCursor int

	// List
	localCols  []string
	remoteCols []string
	cursor     int
	onLocal    bool // true = cursor on left column, false = right
	status     string
}

func New() Model {
	return Model{onLocal: true}
}

func (m *Model) Toggle()      { m.visible = !m.visible }
func (m Model) Visible() bool { return m.visible }

func (m *Model) Show(groupKey string, localCols, remoteCols []string) {
	m.visible = true
	m.localCols = localCols
	m.remoteCols = remoteCols
	m.cursor = 0
	m.onLocal = true
	m.status = ""
	if groupKey == "" {
		m.screen = screenKeyPrompt
		m.keyBuf = ""
		m.keyCursor = 0
	} else {
		m.screen = screenList
	}
}

func (m *Model) SetStatus(s string) { m.status = s }

func (m *Model) SetRemoteCols(cols []string) {
	m.remoteCols = cols
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.screen == screenKeyPrompt {
			return m.updateKeyPrompt(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m Model) updateKeyPrompt(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.keyBuf != "" {
			key := m.keyBuf
			m.screen = screenList
			return m, func() tea.Msg { return KeySavedMsg{Key: key} }
		}
	case "escape", "ctrl+g":
		m.visible = false
		return m, func() tea.Msg { return DismissMsg{} }
	case "backspace":
		if m.keyCursor > 0 {
			m.keyBuf = m.keyBuf[:m.keyCursor-1] + m.keyBuf[m.keyCursor:]
			m.keyCursor--
		}
	case "left":
		if m.keyCursor > 0 {
			m.keyCursor--
		}
	case "right":
		if m.keyCursor < len(m.keyBuf) {
			m.keyCursor++
		}
	default:
		if t := msg.Key().Text; t != "" {
			m.keyBuf = m.keyBuf[:m.keyCursor] + t + m.keyBuf[m.keyCursor:]
			m.keyCursor += len(t)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "escape", "ctrl+g":
		m.visible = false
		return m, func() tea.Msg { return DismissMsg{} }
	case "tab":
		m.onLocal = !m.onLocal
		m.cursor = 0
	case "j", "down":
		max := len(m.localCols)
		if !m.onLocal {
			max = len(m.remoteCols)
		}
		if m.cursor < max-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "p":
		if m.onLocal && m.cursor < len(m.localCols) {
			col := m.localCols[m.cursor]
			m.status = fmt.Sprintf("Pushing %s...", col)
			return m, func() tea.Msg { return PushMsg{Collection: col} }
		}
	case "enter", "l":
		if !m.onLocal && m.cursor < len(m.remoteCols) {
			col := m.remoteCols[m.cursor]
			m.status = fmt.Sprintf("Pulling %s...", col)
			return m, func() tea.Msg { return PullMsg{Collection: col} }
		}
	}
	return m, nil
}

func (m Model) renderKeyBuf() string {
	c := m.keyCursor
	if c > len(m.keyBuf) {
		c = len(m.keyBuf)
	}
	return m.keyBuf[:c] + "\u2588" + m.keyBuf[c:]
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	active := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inactive := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var s string

	if m.screen == screenKeyPrompt {
		s = title.Render("Team Sync - Enter Group Key") + "\n\n"
		s += "Passphrase: " + m.renderKeyBuf() + "\n\n"
		s += dim.Render("Enter to confirm, Esc to cancel")
	} else {
		s = title.Render("Team Sync") + "\n\n"

		// Column headers
		leftHeader := "Local"
		rightHeader := "Remote"
		if m.onLocal {
			leftHeader = active.Render("> Local")
			rightHeader = inactive.Render("  Remote")
		} else {
			leftHeader = inactive.Render("  Local")
			rightHeader = active.Render("> Remote")
		}
		s += leftHeader + "          " + rightHeader + "\n"
		s += dim.Render("─────────────────────────────────") + "\n"

		maxRows := len(m.localCols)
		if len(m.remoteCols) > maxRows {
			maxRows = len(m.remoteCols)
		}

		for i := 0; i < maxRows; i++ {
			leftCol := "               "
			rightCol := ""

			if i < len(m.localCols) {
				prefix := "  "
				if m.onLocal && i == m.cursor {
					prefix = cursorStyle.Render("> ")
				}
				leftCol = fmt.Sprintf("%-15s", prefix+m.localCols[i])
			}

			if i < len(m.remoteCols) {
				prefix := "  "
				if !m.onLocal && i == m.cursor {
					prefix = cursorStyle.Render("> ")
				}
				rightCol = prefix + m.remoteCols[i]
			}

			s += leftCol + " " + rightCol + "\n"
		}

		if m.status != "" {
			s += "\n" + m.status
		}

		s += "\n" + dim.Render("Tab switch column, p push, Enter/l pull, Esc close")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/tui/syncmodal/`
Expected: Success

**Step 3: Commit**

```bash
git add internal/tui/syncmodal/syncmodal.go
git commit -m "feat(tui): add sync modal component"
```

---

### Task 5: Wire sync modal into main.go

**Files:**
- Modify: `cmd/ratatuile/main.go`

**Step 1: Add import and field**

Add to imports:

```go
"github.com/erlandas/ratatuile/internal/tui/syncmodal"
psync "github.com/erlandas/ratatuile/internal/sync"
```

Add field to `model` struct:

```go
syncModal  syncmodal.Model
```

Add to `initialModel`:

```go
syncModal: syncmodal.New(),
```

**Step 2: Add Ctrl+G keybinding to open the modal**

In the `tea.KeyPressMsg` switch (after the `ctrl+i` case), add:

```go
case "ctrl+g":
	groupKey := m.store.LoadGroupKey()
	localCols, _ := m.store.ListCollections(context.Background())
	endpoint := m.store.LoadSyncEndpoint()
	ns := ""
	var remoteCols []string
	if groupKey != "" {
		ns, _ = psync.DeriveKey(groupKey)
		sc := psync.NewClient(endpoint)
		remoteCols, _ = sc.ListRemote(ns)
	}
	m.syncModal.Show(groupKey, localCols, remoteCols)
	return m, nil
```

**Step 3: Add modal input delegation**

In the `tea.KeyPressMsg` handler, after the curlImport visible check, add:

```go
if m.syncModal.Visible() {
	var cmd tea.Cmd
	m.syncModal, cmd = m.syncModal.Update(msg)
	return m, cmd
}
```

**Step 4: Handle sync messages**

Add these cases to the main `Update` switch:

```go
case syncmodal.DismissMsg:
	return m, nil

case syncmodal.KeySavedMsg:
	m.store.SaveGroupKey(msg.Key)
	m.status = "Group key saved"
	// Fetch remote list now that we have a key
	endpoint := m.store.LoadSyncEndpoint()
	ns, _ := psync.DeriveKey(msg.Key)
	sc := psync.NewClient(endpoint)
	remoteCols, _ := sc.ListRemote(ns)
	m.syncModal.SetRemoteCols(remoteCols)
	return m, nil

case syncmodal.PushMsg:
	ctx := context.Background()
	col, err := m.store.LoadCollection(ctx, msg.Collection)
	if err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Error: %v", err))
		return m, nil
	}
	data, _ := json.Marshal(col)
	groupKey := m.store.LoadGroupKey()
	ns, aesKey := psync.DeriveKey(groupKey)
	encrypted, err := psync.Encrypt(aesKey, data)
	if err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Encrypt error: %v", err))
		return m, nil
	}
	endpoint := m.store.LoadSyncEndpoint()
	sc := psync.NewClient(endpoint)
	if err := sc.Push(ns, msg.Collection, encrypted); err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Push error: %v", err))
	} else {
		m.syncModal.SetStatus(fmt.Sprintf("Pushed %s", msg.Collection))
		m.status = fmt.Sprintf("Pushed: %s", msg.Collection)
	}
	return m, nil

case syncmodal.PullMsg:
	groupKey := m.store.LoadGroupKey()
	ns, aesKey := psync.DeriveKey(groupKey)
	endpoint := m.store.LoadSyncEndpoint()
	sc := psync.NewClient(endpoint)
	encrypted, err := sc.Pull(ns, msg.Collection)
	if err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Pull error: %v", err))
		return m, nil
	}
	data, err := psync.Decrypt(aesKey, encrypted)
	if err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Decrypt error: %v", err))
		return m, nil
	}
	var col domain.Collection
	if err := json.Unmarshal(data, &col); err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Parse error: %v", err))
		return m, nil
	}
	ctx := context.Background()
	if err := m.store.SaveCollection(ctx, col); err != nil {
		m.syncModal.SetStatus(fmt.Sprintf("Save error: %v", err))
	} else {
		m.syncModal.SetStatus(fmt.Sprintf("Pulled %s", msg.Collection))
		m.status = fmt.Sprintf("Pulled: %s", msg.Collection)
		// Reload tree with updated collections
		names, _ := m.store.ListCollections(ctx)
		var collections []domain.Collection
		for _, n := range names {
			c, err := m.store.LoadCollection(ctx, n)
			if err == nil {
				collections = append(collections, c)
			}
		}
		m.tree = tree.New(collections)
		m.tree.SetFocused(m.focusedPane == paneTree)
		m.layoutPanes()
	}
	return m, nil
```

**Step 5: Add sync modal to View**

After the newReq overlay block, add:

```go
if m.syncModal.Visible() {
	overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.syncModal.View())
	v := tea.NewView(overlay)
	v.AltScreen = true
	return v
}
```

**Step 6: Add `encoding/json` to imports** (if not already there)

**Step 7: Verify it compiles**

Run: `go build ./cmd/ratatuile/`
Expected: Success

**Step 8: Commit**

```bash
git add cmd/ratatuile/main.go
git commit -m "feat: wire sync modal into main TUI (Ctrl+G)"
```

---

### Task 6: Cloudflare Worker

**Files:**
- Create: `worker/sync/wrangler.toml`
- Create: `worker/sync/src/index.js`

**Step 1: Create wrangler config**

```toml
# worker/sync/wrangler.toml
name = "ratatuile-sync"
main = "src/index.js"
compatibility_date = "2024-01-01"

[[kv_namespaces]]
binding = "SYNC"
id = "<create via wrangler kv:namespace create SYNC>"
```

**Step 2: Create the worker**

```js
// worker/sync/src/index.js
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const parts = url.pathname.split("/").filter(Boolean);

    const headers = {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, PUT, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type",
    };

    if (request.method === "OPTIONS") {
      return new Response(null, { status: 204, headers });
    }

    // GET /:namespace — list collections
    if (request.method === "GET" && parts.length === 1) {
      const namespace = parts[0];
      const list = await env.SYNC.list({ prefix: namespace + "/" });
      const names = list.keys.map((k) => k.name.slice(namespace.length + 1));
      return new Response(JSON.stringify(names), {
        headers: { ...headers, "Content-Type": "application/json" },
      });
    }

    // GET /:namespace/:collection — pull
    if (request.method === "GET" && parts.length === 2) {
      const key = parts.join("/");
      const value = await env.SYNC.get(key, { type: "arrayBuffer" });
      if (value === null) {
        return new Response("not found", { status: 404, headers });
      }
      return new Response(value, {
        headers: { ...headers, "Content-Type": "application/octet-stream" },
      });
    }

    // PUT /:namespace/:collection — push
    if (request.method === "PUT" && parts.length === 2) {
      const key = parts.join("/");
      const body = await request.arrayBuffer();
      await env.SYNC.put(key, body);
      return new Response("ok", { headers });
    }

    return new Response("bad request", { status: 400, headers });
  },
};
```

**Step 3: Commit**

```bash
git add worker/sync/
git commit -m "feat: add Cloudflare Worker for sync KV storage"
```

---

### Task 7: Update help overlay and status bar hints

**Files:**
- Modify: `internal/tui/help/help.go`
- Modify: `cmd/ratatuile/main.go` (status bar hints)

**Step 1: Add Ctrl+G to help bindings**

In `help.go`, add to the `bindings` slice:

```go
{"Ctrl+G", "Team sync (push/pull collections)"},
```

**Step 2: Add hint to status bar in main.go View()**

Add after the `"?"` hint:

```go
hintSep + hintKey.Render("^G") + hintDesc.Render(" sync")
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/ratatuile/`
Expected: Success

**Step 4: Commit**

```bash
git add internal/tui/help/help.go cmd/ratatuile/main.go
git commit -m "feat: add sync keybinding to help overlay and status bar"
```

---

### Task 8: End-to-end manual test

**Steps:**
1. Run: `go run ./cmd/ratatuile/`
2. Press `Ctrl+G` — should see "Enter Group Key" prompt
3. Type a passphrase, press Enter — key saved, list screen appears
4. (Worker must be deployed for push/pull to work against real endpoint)
5. Verify Esc closes the modal
6. Verify `~/.ratatuile/group_key` contains the passphrase
7. Run: `go test ./...` — all tests pass

Run: `go test ./... && go build ./cmd/ratatuile/`
Expected: All tests pass, binary builds
