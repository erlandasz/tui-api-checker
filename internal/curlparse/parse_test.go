package curlparse

import (
	"testing"
)

func TestParseSimpleGET(t *testing.T) {
	req, err := Parse(`curl https://api.example.com/users`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "GET" {
		t.Errorf("method = %q, want GET", req.Method)
	}
	if req.URL != "https://api.example.com/users" {
		t.Errorf("url = %q, want https://api.example.com/users", req.URL)
	}
	if req.Name != "users" {
		t.Errorf("name = %q, want users", req.Name)
	}
}

func TestParseWithHeaders(t *testing.T) {
	req, err := Parse(`curl -X POST -H 'Content-Type: application/json' -H "Authorization: Bearer tok" https://api.example.com/login`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q", req.Headers["Content-Type"])
	}
	if req.Headers["Authorization"] != "Bearer tok" {
		t.Errorf("Authorization = %q", req.Headers["Authorization"])
	}
}

func TestParseWithBody(t *testing.T) {
	req, err := Parse(`curl -d '{"name":"test"}' https://api.example.com/items`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST (implied by -d)", req.Method)
	}
	if req.Body != `{"name":"test"}` {
		t.Errorf("body = %q", req.Body)
	}
}

func TestParseWithQueryParams(t *testing.T) {
	req, err := Parse(`curl 'https://api.example.com/search?q=hello&limit=10'`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Params["q"] != "hello" {
		t.Errorf("param q = %q, want hello", req.Params["q"])
	}
	if req.Params["limit"] != "10" {
		t.Errorf("param limit = %q, want 10", req.Params["limit"])
	}
	if req.URL != "https://api.example.com/search" {
		t.Errorf("url = %q, want without query string", req.URL)
	}
}

func TestParseMultiline(t *testing.T) {
	curl := "curl -X PUT \\\n  -H 'Accept: application/json' \\\n  'https://api.example.com/items/42'"
	req, err := Parse(curl)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "PUT" {
		t.Errorf("method = %q, want PUT", req.Method)
	}
	if req.Headers["Accept"] != "application/json" {
		t.Errorf("Accept = %q", req.Headers["Accept"])
	}
	if req.Name != "42" {
		t.Errorf("name = %q, want 42", req.Name)
	}
}

func TestParseRootPath(t *testing.T) {
	req, err := Parse(`curl https://example.com/`)
	if err != nil {
		t.Fatal(err)
	}
	if req.Name != "example.com" {
		t.Errorf("name = %q, want example.com for root path", req.Name)
	}
}
