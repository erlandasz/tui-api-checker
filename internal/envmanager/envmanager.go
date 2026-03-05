package envmanager

import (
	"strings"

	"github.com/erlandas/postmaniux/internal/domain"
)

// Resolve replaces {{key}} placeholders with values from vars.
// Unresolved placeholders are left as-is.
func Resolve(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

// ResolveRequest returns a copy of req with all {{var}} placeholders
// replaced using the environment's variables.
func ResolveRequest(req domain.Request, env domain.Environment) domain.Request {
	resolved := req
	resolved.URL = Resolve(req.URL, env.Variables)
	resolved.Body = Resolve(req.Body, env.Variables)

	if len(req.Headers) > 0 {
		resolved.Headers = make(map[string]string, len(req.Headers))
		for k, v := range req.Headers {
			resolved.Headers[k] = Resolve(v, env.Variables)
		}
	}

	if len(req.Params) > 0 {
		resolved.Params = make(map[string]string, len(req.Params))
		for k, v := range req.Params {
			resolved.Params[k] = Resolve(v, env.Variables)
		}
	}

	return resolved
}
