package server

import "testing"

func TestParseTailscaleStatus(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{name: "connected", json: `{"BackendState":"Running"}`, want: "Running"},
		{name: "logged out", json: `{"BackendState":"Stopped"}`, want: "Stopped"},
		{name: "missing state", json: `{}`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := parseTailscaleStatus([]byte(tt.json))
			if err != nil || status.BackendState != tt.want {
				t.Fatalf("parseTailscaleStatus() = %#v, %v; want state %q", status, err, tt.want)
			}
		})
	}
}

func TestParseTailscaleStatusRejectsMalformedJSON(t *testing.T) {
	if _, err := parseTailscaleStatus([]byte(`{`)); err == nil {
		t.Fatal("parseTailscaleStatus() succeeded for malformed JSON")
	}
}
