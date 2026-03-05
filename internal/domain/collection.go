package domain

// Collection groups requests under a name.
type Collection struct {
	Name     string    `json:"name"`
	Requests []Request `json:"requests,omitempty"`
}
