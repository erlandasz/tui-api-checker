package storage

import (
	"context"
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
