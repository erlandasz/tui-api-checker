package envmanager

import (
	"strings"
	"time"

	"github.com/erlandas/postmaniux/internal/domain"
)

// DateVars returns built-in date variables computed from now.
func DateVars(now time.Time) map[string]string {
	weekday := now.Weekday()
	daysSinceMonday := int(weekday+6) % 7 // Monday=0
	startOfWeek := now.AddDate(0, 0, -daysSinceMonday)
	endOfWeek := startOfWeek.AddDate(0, 0, 6)

	y, month, _ := now.Date()
	startOfMonth := time.Date(y, month, 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	f := "2006-01-02"
	return map[string]string{
		"$today":        now.Format(f),
		"$yesterday":    now.AddDate(0, 0, -1).Format(f),
		"$tomorrow":     now.AddDate(0, 0, 1).Format(f),
		"$startOfWeek":  startOfWeek.Format(f),
		"$endOfWeek":    endOfWeek.Format(f),
		"$startOfMonth": startOfMonth.Format(f),
		"$endOfMonth":   endOfMonth.Format(f),
	}
}

// KnownVars returns the set of variable names that would resolve,
// combining date vars and the given environment variables.
func KnownVars(envVars map[string]string) map[string]bool {
	known := make(map[string]bool)
	for k := range DateVars(time.Now()) {
		known[k] = true
	}
	for k := range envVars {
		known[k] = true
	}
	return known
}

// Resolve replaces {{key}} placeholders with values from vars.
// Built-in date variables ({{$today}}, etc.) are resolved first,
// then user vars override them. Unresolved placeholders are left as-is.
func Resolve(s string, vars map[string]string) string {
	// Apply date vars first
	for k, v := range DateVars(time.Now()) {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	// User vars override
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
