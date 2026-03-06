package envmanager

import (
	"testing"
	"time"

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

func TestDateVars(t *testing.T) {
	// Thursday 2026-03-06
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	vars := DateVars(now)

	tests := []struct {
		key  string
		want string
	}{
		{"$today", "2026-03-06"},
		{"$yesterday", "2026-03-05"},
		{"$tomorrow", "2026-03-07"},
		{"$startOfWeek", "2026-03-02"},  // Monday
		{"$endOfWeek", "2026-03-08"},    // Sunday
		{"$startOfMonth", "2026-03-01"},
		{"$endOfMonth", "2026-03-31"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := vars[tt.key]
			if got != tt.want {
				t.Errorf("DateVars()[%q] = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestDateVarsMonday(t *testing.T) {
	// Monday 2026-03-02
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	vars := DateVars(now)

	if vars["$startOfWeek"] != "2026-03-02" {
		t.Errorf("startOfWeek on Monday = %q, want 2026-03-02", vars["$startOfWeek"])
	}
	if vars["$endOfWeek"] != "2026-03-08" {
		t.Errorf("endOfWeek on Monday = %q, want 2026-03-08", vars["$endOfWeek"])
	}
}

func TestDateVarsSunday(t *testing.T) {
	// Sunday 2026-03-08
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	vars := DateVars(now)

	if vars["$startOfWeek"] != "2026-03-02" {
		t.Errorf("startOfWeek on Sunday = %q, want 2026-03-02", vars["$startOfWeek"])
	}
	if vars["$endOfWeek"] != "2026-03-08" {
		t.Errorf("endOfWeek on Sunday = %q, want 2026-03-08", vars["$endOfWeek"])
	}
}

func TestResolveDateVars(t *testing.T) {
	s := Resolve("from={{$today}}&key={{api_key}}", map[string]string{"api_key": "abc"})
	if s == "from={{$today}}&key=abc" {
		t.Error("$today was not resolved")
	}
	if s[len("from="):len("from=")+10] != time.Now().Format("2006-01-02") {
		t.Errorf("unexpected date in %q", s)
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
