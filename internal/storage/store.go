package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erlandas/postmaniux/internal/domain"
)

// FileStore persists collections and environments as JSON files.
// Root dir is injected via constructor for testability.
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

func (s *FileStore) activeEnvPath() string {
	return filepath.Join(s.root, "active_env")
}

func (s *FileStore) SaveActiveEnv(name string) error {
	if err := os.MkdirAll(s.root, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.activeEnvPath(), []byte(name), 0644)
}

func (s *FileStore) LoadActiveEnv() string {
	data, err := os.ReadFile(s.activeEnvPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
