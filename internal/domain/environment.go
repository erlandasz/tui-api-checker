package domain

// Environment holds named variables for template substitution.
type Environment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}
