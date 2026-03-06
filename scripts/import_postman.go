// One-off script to import a Postman data export into postmaniux storage format.
//
// Usage:
//
//	go run scripts/import_postman.go /path/to/postman-export-dir
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Postman collection types

type postmanCollection struct {
	Info struct {
		Name string `json:"name"`
	} `json:"info"`
	Item []postmanItem `json:"item"`
}

type postmanItem struct {
	Name    string        `json:"name"`
	Item    []postmanItem `json:"item,omitempty"`
	Request *postmanReq   `json:"request,omitempty"`
}

type postmanReq struct {
	Method string          `json:"method"`
	Header []postmanHeader `json:"header"`
	URL    postmanURL      `json:"url"`
	Body   *postmanBody    `json:"body,omitempty"`
}

type postmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanURL struct {
	Raw   string         `json:"raw"`
	Query []postmanQuery `json:"query,omitempty"`
}

type postmanQuery struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

// Postman environment types

type postmanEnv struct {
	Name   string            `json:"name"`
	Values []postmanEnvValue `json:"values"`
}

type postmanEnvValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// postmaniux domain types

type request struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type collection struct {
	Name     string    `json:"name"`
	Requests []request `json:"requests,omitempty"`
}

type environment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: go run scripts/import_postman.go <postman-export-dir>\n")
		os.Exit(1)
	}
	exportDir := os.Args[1]

	home, err := os.UserHomeDir()
	if err != nil {
		fatal("getting home dir: %v", err)
	}
	outRoot := filepath.Join(home, ".postmaniux")

	// Import collections
	collDir := filepath.Join(exportDir, "collection")
	if entries, err := os.ReadDir(collDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			path := filepath.Join(collDir, e.Name())
			if err := importCollection(path, outRoot); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping collection %s: %v\n", e.Name(), err)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "no collection/ directory found, skipping collections\n")
	}

	// Import environments
	envDir := filepath.Join(exportDir, "environment")
	if entries, err := os.ReadDir(envDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			path := filepath.Join(envDir, e.Name())
			if err := importEnvironment(path, outRoot); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping environment %s: %v\n", e.Name(), err)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "no environment/ directory found, skipping environments\n")
	}
}

func importCollection(path, outRoot string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var pc postmanCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	var reqs []request
	flattenItems(pc.Item, nil, &reqs)

	col := collection{
		Name:     pc.Info.Name,
		Requests: reqs,
	}

	outDir := filepath.Join(outRoot, "collections", col.Name)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	outPath := filepath.Join(outDir, "collection.json")
	if err := writeJSON(outPath, col); err != nil {
		return err
	}
	fmt.Printf("collection: %s (%d requests) -> %s\n", col.Name, len(reqs), outPath)
	return nil
}

func flattenItems(items []postmanItem, prefix []string, out *[]request) {
	for _, item := range items {
		if item.Item != nil {
			flattenItems(item.Item, append(prefix, item.Name), out)
			continue
		}
		if item.Request == nil {
			continue
		}
		r := convertRequest(item.Name, prefix, item.Request)
		*out = append(*out, r)
	}
}

func convertRequest(name string, prefix []string, pr *postmanReq) request {
	fullName := name
	if len(prefix) > 0 {
		fullName = strings.Join(prefix, " / ") + " / " + name
	}

	headers := make(map[string]string)
	for _, h := range pr.Header {
		headers[h.Key] = h.Value
	}

	params := make(map[string]string)
	for _, q := range pr.URL.Query {
		if !q.Disabled {
			params[q.Key] = q.Value
		}
	}

	var body string
	if pr.Body != nil && pr.Body.Raw != "" {
		body = pr.Body.Raw
	}

	r := request{
		Name:   fullName,
		Method: pr.Method,
		URL:    pr.URL.Raw,
	}
	if len(headers) > 0 {
		r.Headers = headers
	}
	if len(params) > 0 {
		r.Params = params
	}
	if body != "" {
		r.Body = body
	}
	return r
}

func importEnvironment(path, outRoot string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var pe postmanEnv
	if err := json.Unmarshal(data, &pe); err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	vars := make(map[string]string)
	for _, v := range pe.Values {
		if v.Enabled && v.Value != "" {
			vars[v.Key] = v.Value
		}
	}

	env := environment{
		Name:      pe.Name,
		Variables: vars,
	}

	envDir := filepath.Join(outRoot, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return err
	}
	outPath := filepath.Join(envDir, env.Name+".json")
	if err := writeJSON(outPath, env); err != nil {
		return err
	}
	fmt.Printf("environment: %s (%d variables) -> %s\n", env.Name, len(vars), outPath)
	return nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fatal: "+format+"\n", args...)
	os.Exit(1)
}
