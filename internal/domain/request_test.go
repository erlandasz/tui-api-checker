package domain

import "testing"

func TestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     Request
		wantErr bool
	}{
		{"valid GET", Request{Name: "test", Method: "GET", URL: "http://localhost"}, false},
		{"empty name", Request{Name: "", Method: "GET", URL: "http://localhost"}, true},
		{"empty URL", Request{Name: "test", Method: "GET", URL: ""}, true},
		{"empty method", Request{Name: "test", Method: "", URL: "http://localhost"}, true},
		{"valid POST", Request{Name: "test", Method: "POST", URL: "http://localhost", Body: `{"a":1}`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
