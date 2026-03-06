package curlparse

import (
	"net/url"
	"strings"

	"github.com/erlandas/ratatuile/internal/domain"
)

// Parse converts a curl command string into a domain.Request.
// It handles -X, -H, -d/--data, and the URL argument.
func Parse(curl string) (domain.Request, error) {
	args := tokenize(curl)
	req := domain.Request{
		Method:  "GET",
		Headers: make(map[string]string),
		Params:  make(map[string]string),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "curl":
			continue
		case arg == "-X" || arg == "--request":
			if i+1 < len(args) {
				i++
				req.Method = strings.ToUpper(args[i])
			}
		case arg == "-H" || arg == "--header":
			if i+1 < len(args) {
				i++
				if k, v, ok := strings.Cut(args[i], ":"); ok {
					req.Headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
		case arg == "-d" || arg == "--data" || arg == "--data-raw" || arg == "--data-binary":
			if i+1 < len(args) {
				i++
				req.Body = args[i]
				if req.Method == "GET" {
					req.Method = "POST"
				}
			}
		case strings.HasPrefix(arg, "-"):
			// Skip unknown flags; consume next arg if it looks like a value
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
		default:
			// Positional argument = URL
			req.URL = arg
		}
	}

	// Extract query params from URL
	if req.URL != "" {
		if u, err := url.Parse(req.URL); err == nil && u.RawQuery != "" {
			for k, vals := range u.Query() {
				if len(vals) > 0 {
					req.Params[k] = vals[0]
				}
			}
			u.RawQuery = ""
			u.ForceQuery = false
			req.URL = u.String()
		}
	}

	// Derive a name from the URL path
	if req.URL != "" {
		if u, err := url.Parse(req.URL); err == nil {
			path := strings.TrimRight(u.Path, "/")
			if path == "" {
				req.Name = u.Host
			} else {
				segments := strings.Split(path, "/")
				req.Name = segments[len(segments)-1]
			}
		}
	}

	return req, nil
}

// tokenize splits a curl command into arguments, respecting single/double quotes
// and backslash-escaped line continuations.
func tokenize(s string) []string {
	// Normalize line continuations
	s = strings.ReplaceAll(s, "\\\n", " ")
	s = strings.ReplaceAll(s, "\\\r\n", " ")

	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
