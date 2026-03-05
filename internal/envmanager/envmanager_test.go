package envmanager

import (
	"testing"

	"github.com/erlandas/postmaniux/internal/domain"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
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
