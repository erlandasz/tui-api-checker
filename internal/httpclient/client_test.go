package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erlandas/postmaniux/internal/domain"
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
