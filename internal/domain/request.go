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
